#!/usr/bin/env bash
# Runs a Salus plan end to end: one headless Claude Code session per work package (each commits once),
# then pushes and opens pull requests.
#
# Usage:  bash scripts/run-feature.sh docs/alati [--from 1.2] [--only 2.1] [--no-pr] [--stacked]
#         bash scripts/run-feature.sh docs/payments --restack
#         bash scripts/run-feature.sh docs/payments --unhold     # also run the rows marked "hold"
#
# Reads <plan>/RUN.tsv (package, tickets, title, and an optional fourth column "hold").
# One PR (default): every package on branch feat/<prefix> off origin/<base>, one PR for the whole feature.
# Stacked (--stacked, or "# stack: yes" in RUN.tsv): the plan and every package get their own branch,
#   stack/<prefix>/plan, stack/<prefix>/<package>, each cut from the one before, and their own PR whose
#   base is the branch below. Copilot is asked to review each PR.
# --restack: after the lowest PRs were squash-merged (or a lower branch got new commits), rebase the open
#   branches onto what is below them, force-push them with a lease, and retarget their PRs. Runs nothing.
# A row marked "hold" is not run, and the run stops there, unless --unhold is given.
# Safe to re-run: packages whose "<prefix>(<package>):" commit is already on their branch are skipped.
# Optional env: JIRA_EMAIL, JIRA_API_TOKEN (Atlassian API token), JIRA_BASE_URL (https://<site>.atlassian.net).
set -euo pipefail

usage() { sed -n '2,20p' "$0"; exit 2; }
die() { echo "$*" >&2; exit 1; }
[ $# -ge 1 ] || usage
plan="${1%/}"; shift
from=""; only=""; make_pr=1; stacked=""; restack=0; unhold=0
while [ $# -gt 0 ]; do
  case "$1" in
    --from) from="$2"; shift 2 ;;
    --only) only="$2"; shift 2 ;;
    --no-pr) make_pr=0; shift ;;
    --stacked) stacked=1; shift ;;
    --restack) restack=1; stacked=1; shift ;;
    --unhold) unhold=1; shift ;;
    *) usage ;;
  esac
done

root=$(git rev-parse --show-toplevel); cd "$root"
manifest="$plan/RUN.tsv"
[ -f "$manifest" ] || { echo "missing $manifest"; exit 2; }
# The run switches branches, and a package may edit the manifest: read it once, as it is now.
manifest_copy=$(mktemp); trap 'rm -f "$manifest_copy"' EXIT
cp "$manifest" "$manifest_copy"; manifest="$manifest_copy"
[ -f .claude/agents/package-runner.md ] || { echo "missing .claude/agents/package-runner.md"; exit 2; }
meta() { awk -v k="$1" 'index($0, "# " k ":") == 1 { sub("^# " k ": *", ""); print; exit }' "$manifest"; }
prefix=$(meta prefix); epic=$(meta epic); title=$(meta title); base=$(meta base); base=${base:-main}
[ -n "$prefix" ] && [ -n "$title" ] || { echo "$plan/RUN.tsv needs '# prefix:' and '# title:' lines"; exit 2; }
if [ -z "$stacked" ]; then
  case "$(meta stack)" in yes|true|1) stacked=1 ;; *) stacked=0 ;; esac
fi
runs=".claude/runs/$prefix"
mkdir -p "$runs"

for cmd in git gh jq curl; do
  command -v "$cmd" >/dev/null || { echo "needs $cmd on PATH"; exit 2; }
done
[ "$restack" = 1 ] || command -v claude >/dev/null || { echo "needs claude on PATH"; exit 2; }
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo "the working tree has uncommitted changes; commit or move them first"; exit 2
fi

# The manifest's rows: package, tickets, title, hold. Comments and blank lines are skipped.
rows() {
  awk -F'\t' -v unhold="$unhold" '!/^#/ && NF { h = $4; if (unhold == 1) h = ""; print $1 "\t" $2 "\t" $3 "\t" h }' "$manifest"
}
packages() { rows | awk -F'\t' '$4 != "hold" { print $1 }'; }

# Jira automation closes every key it finds in a merged PR. A ticket's key may appear only in the PR of
# the last package that lists it (hold rows included), so merging an earlier package closes nothing.
owned_keys() {
  local pkg="$1"
  rows | awk -F'\t' -v p="$pkg" -v epic="$epic" '
    { n = split($2, k, ","); for (i = 1; i <= n; i++) if (k[i] != "") last[k[i]] = $1; lastrow = $1 }
    END {
      if (epic != "" && !(epic in last)) last[epic] = lastrow
      for (key in last) if (last[key] == p) print key
    }' | sort | tr '\n' ' ' | sed 's/ $//'
}
strip_keys() { sed -E 's/\b([A-Z][A-Z0-9]+)-([0-9]+)\b/\1 \2/g'; }

jira_ticket() {
  local key="$1"
  [ -n "${JIRA_EMAIL:-}" ] && [ -n "${JIRA_API_TOKEN:-}" ] && [ -n "${JIRA_BASE_URL:-}" ] || return 1
  curl -fsS -u "$JIRA_EMAIL:$JIRA_API_TOKEN" \
    "${JIRA_BASE_URL}/rest/api/2/issue/$key?fields=summary,status,description,comment" |
    jq -r --arg k "$key" '
      "## \($k): \(.fields.summary)\nStatus: \(.fields.status.name)\n\n\(.fields.description // "(no description)")\n"
      + ([.fields.comment.comments[]? | "\n### Comment, \(.created[0:10])\n\(.body)\n"] | join(""))'
}

# run_package <package> <tickets> <title> <allowed keys>: one headless session that must leave one commit.
run_package() {
  local pkg="$1" tickets="$2" name="$3" allowed="$4"
  local ticketfile="$runs/$pkg.tickets.md" report="$runs/$pkg.report.md"
  : > "$ticketfile"
  for key in ${tickets//,/ }; do
    jira_ticket "$key" >> "$ticketfile" 2>/dev/null ||
      printf '## %s\n(Jira was not reachable; work from the plan docs.)\n\n' "$key" >> "$ticketfile"
  done
  rm -f "$report"
  echo "== $prefix $pkg: $name ($(date '+%H:%M'))"
  local prompt="Read .claude/agents/package-runner.md and follow it exactly.
Plan: $plan
Package: $pkg ($name)
Commit prefix: $prefix
Tickets file: $ticketfile
Report path: $report"
  if [ "$stacked" = 1 ]; then
    prompt="$prompt
Jira keys allowed in the commit message: ${allowed:-none}"
  fi
  if ! claude -p "$prompt" \
      --permission-mode dontAsk \
      --allowedTools "Read,Edit,Write,Grep,Glob,Bash,Agent,WebFetch,WebSearch" \
      --output-format json < /dev/null > "$runs/$pkg.session.json"; then
    die "the session for $pkg ended with an error; see $runs/$pkg.session.json"
  fi
}

check_committed() {
  local pkg="$1" range="$2" report="$runs/$1.report.md"
  git log --format=%s "$range" | grep -q "^$prefix($pkg):" || die "no '$prefix($pkg):' commit; stopping. Report: $report"
  [ -z "$(git status --porcelain --untracked-files=no)" ] || die "the session left uncommitted changes after $pkg; stopping"
  echo "   committed $(git log -1 --format='%h %s')"
}

# ---------------------------------------------------------------------------------------------------
# One branch, one PR: the original flow.
# ---------------------------------------------------------------------------------------------------
single_pr() {
  local branch="feat/$prefix"
  git fetch --quiet origin "$base"
  if git show-ref --verify --quiet "refs/heads/$branch"; then
    git switch --quiet "$branch"
  else
    git switch --quiet -c "$branch" "origin/$base"
  fi
  if [ -z "$(git ls-files "$plan/SPEC.md")" ]; then
    git add "$plan"
    git commit --quiet -m "$prefix(plan): $title specification and implementation plan"
  fi

  done_pkg() { git log --format=%s "origin/$base..HEAD" | grep -q "^$prefix($1):"; }

  local started=0 pkg tickets name hold
  [ -z "$from" ] && started=1
  while IFS=$'\t' read -r pkg tickets name hold; do
    [ "$pkg" = "$from" ] && started=1
    [ "$started" = 1 ] || continue
    if [ -n "$only" ] && [ "$pkg" != "$only" ]; then continue; fi
    if [ "$hold" = "hold" ]; then echo "== $prefix $pkg is on hold; stopping here"; break; fi
    if done_pkg "$pkg"; then echo "== $prefix $pkg already committed, skipping"; continue; fi
    run_package "$pkg" "$tickets" "$name" ""
    check_committed "$pkg" "origin/$base..HEAD"
  done < <(rows)

  [ "$make_pr" = 1 ] || { echo "done (no PR requested)"; exit 0; }
  local missing
  missing=$(packages | while read -r p; do done_pkg "$p" || echo "$p"; done | tr '\n' ' ')
  if [ -n "${missing// /}" ]; then
    echo "not opening a PR yet; packages without a commit: $missing"; exit 0
  fi

  local keys body="$runs/PR_BODY.md"
  keys=$( { echo "$epic"; packages | while read -r p; do rows | awk -F'\t' -v p="$p" '$1 == p { print $2 }'; done | tr ',' '\n'; } |
    grep -v '^$' | sort -u | tr '\n' ' ')
  {
    echo "## $title"
    echo
    echo "Built package by package from \`$plan/\` with \`scripts/run-feature.sh\`: one Claude Code session and one commit per package. Plan, decisions and the running record: \`$plan/PROGRESS.md\`."
    echo
    echo "### Commits"
    echo
    git log --reverse --format='- %h %s' "origin/$base..HEAD" | strip_keys
    echo
    echo "### Package reports"
    for p in $(packages); do
      local f="$runs/$p.report.md"
      [ -f "$f" ] || continue
      echo
      echo "<details><summary>$p</summary>"
      echo
      strip_keys < "$f"
      echo
      echo "</details>"
    done
    echo
    echo "### Jira"
    echo
    echo "$keys"
    echo
    echo "🤖 Generated with [Claude Code](https://claude.com/claude-code)"
  } > "$body"

  git push --quiet -u origin "$branch"
  if gh pr view "$branch" >/dev/null 2>&1; then
    gh pr edit "$branch" --body-file "$body" >/dev/null
  else
    gh pr create --base "$base" --head "$branch" --title "$title ($epic)" --body-file "$body" >/dev/null
  fi
  echo "PR: $(gh pr view "$branch" --json url -q .url)"
}

# ---------------------------------------------------------------------------------------------------
# Stacked: one branch and one PR per package, each on top of the one before.
# ---------------------------------------------------------------------------------------------------
stack_branch() { echo "stack/$prefix/$1"; }

# The stack, bottom first: the plan branch, then every package that is not on hold.
chain() { echo plan; packages; }

pr_json() { gh pr view "$1" --json number,state,url,baseRefName 2>/dev/null || true; }

# ensure_local <branch>: create the local branch from origin if it only exists there, and fast-forward it
# when origin is ahead (a suggestion committed on GitHub). A branch that diverged from origin stops the run.
ensure_local() {
  local b="$1"
  local has_remote=0
  git show-ref --verify --quiet "refs/remotes/origin/$b" && has_remote=1
  if ! git show-ref --verify --quiet "refs/heads/$b"; then
    [ "$has_remote" = 1 ] || return 1
    git branch --quiet "$b" "origin/$b"
    return 0
  fi
  [ "$has_remote" = 1 ] || return 0
  local l r
  l=$(git rev-parse "refs/heads/$b"); r=$(git rev-parse "refs/remotes/origin/$b")
  [ "$l" = "$r" ] && return 0
  if git merge-base --is-ancestor "$l" "$r"; then
    if [ "$(git branch --show-current)" = "$b" ]; then git merge --quiet --ff-only "origin/$b"; else git branch --quiet -f "$b" "origin/$b"; fi
  elif ! git merge-base --is-ancestor "$r" "$l"; then
    die "$b and origin/$b have diverged; reconcile them by hand, then run again"
  fi
}

request_copilot() {
  local url="$1"
  if gh pr edit "$url" --add-reviewer @copilot >/dev/null 2>&1; then return 0; fi
  local number repo
  number=$(gh pr view "$url" --json number -q .number 2>/dev/null || true)
  repo=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
  if [ -n "$number" ] && [ -n "$repo" ] &&
     gh api -X POST "repos/$repo/pulls/$number/requested_reviewers" -f 'reviewers[]=copilot-pull-request-reviewer[bot]' >/dev/null 2>&1; then
    return 0
  fi
  echo "   could not request a Copilot review on $url; request it on GitHub, or turn on the ruleset's automatic Copilot review" >&2
}

# open_pr <package|plan> <branch> <base branch> <position>: push the branch and open or update its PR.
open_pr() {
  local pkg="$1" b="$2" parent="$3" position="$4"
  local name keys body="$runs/$pkg.PR_BODY.md" report="$runs/$pkg.report.md" parent_pr
  if [ "$pkg" = plan ]; then
    name="specification and implementation plan"; keys=""
  else
    name=$(rows | awk -F'\t' -v p="$pkg" '$1 == p { print $3 }')
    keys=$(owned_keys "$pkg")
  fi
  local range_base="$parent"
  [ "$parent" = "$base" ] && range_base="origin/$base"
  parent_pr=""
  [ "$parent" = "$base" ] || parent_pr=$(pr_json "$parent" | jq -r '.number // empty' 2>/dev/null || true)
  {
    echo "## $prefix $pkg: $name"
    echo
    echo "Part $position of a stacked series built from \`$plan/\` with \`scripts/run-feature.sh --stacked\`: one package, one commit, one PR. $(
      if [ -n "$parent_pr" ]; then echo "Stacked on #$parent_pr (\`$parent\`)."; else echo "Base: \`$parent\`."; fi)"
    echo "Review only this PR's diff. Merge bottom-up with squash, then run \`bash scripts/run-feature.sh $plan --restack\`."
    echo
    echo "### Commits"
    echo
    git log --reverse --format='- %h %s' "$range_base..$b" | strip_keys
    if [ -f "$report" ] && [ "$pkg" != plan ]; then
      echo
      echo "### Package report"
      echo
      strip_keys < "$report"
    fi
    echo
    echo "### Jira"
    echo
    if [ -n "$keys" ]; then echo "$keys"; else echo "None closed by this PR (see the plan's \`RUN.tsv\` for where each ticket closes)."; fi
    echo
    echo "🤖 Generated with [Claude Code](https://claude.com/claude-code)"
  } > "$body"

  git push --quiet -u origin "$b"
  local pr title_line url
  pr=$(pr_json "$b")
  title_line="$prefix $pkg: $name"
  [ -n "$keys" ] && title_line="$title_line (${keys// /, })"
  if [ -n "$pr" ] && [ "$(jq -r .state <<<"$pr")" = OPEN ]; then
    gh pr edit "$b" --base "$parent" --body-file "$body" >/dev/null
    url=$(jq -r .url <<<"$pr")
    echo "   PR updated: $url"
  else
    url=$(gh pr create --base "$parent" --head "$b" --title "$title_line" --body-file "$body")
    echo "   PR opened: $url"
    request_copilot "$url"
  fi
}

stacked_run() {
  git fetch --quiet origin
  local b parent pr

  # The plan branch, off the latest base, unless its PR is already merged.
  b=$(stack_branch plan)
  parent="$base"
  if [ "$(pr_json "$b" | jq -r '.state // empty' 2>/dev/null)" = MERGED ]; then
    echo "== $prefix plan is merged"
  else
    ensure_local "$b" || git branch --quiet "$b" "origin/$base"
    git switch --quiet "$b"
    if [ -z "$(git ls-files "$plan/SPEC.md")" ]; then
      git add "$plan"
      git commit --quiet -m "$prefix(plan): $title specification and implementation plan"
    fi
    if [ "$make_pr" = 1 ]; then open_pr plan "$b" "$base" 1; fi
    parent="$b"
  fi

  local started=0 position=1 pkg tickets name hold
  [ -z "$from" ] && started=1
  while IFS=$'\t' read -r pkg tickets name hold; do
    if [ "$hold" = "hold" ]; then echo "== $prefix $pkg is on hold; stopping here"; break; fi
    position=$((position + 1))
    b=$(stack_branch "$pkg")
    [ "$pkg" = "$from" ] && started=1
    local selected=1
    [ "$started" = 1 ] || selected=0
    if [ -n "$only" ] && [ "$pkg" != "$only" ]; then selected=0; fi

    pr=$(pr_json "$b")
    if [ -n "$pr" ] && [ "$(jq -r .state <<<"$pr")" = MERGED ]; then
      echo "== $prefix $pkg is merged"; parent="$base"; continue
    fi
    local parent_ref="$parent"
    [ "$parent" = "$base" ] && parent_ref="origin/$base"
    if ! ensure_local "$b"; then
      if [ "$selected" = 0 ]; then
        echo "== $prefix $pkg has no branch yet; stopping (the packages above it build on it)"; break
      fi
      git branch --quiet "$b" "$parent_ref"
    fi
    git switch --quiet "$b"
    git merge-base --is-ancestor "$parent_ref" "$b" ||
      die "$b is not on top of $parent; run 'bash scripts/run-feature.sh $plan --restack' first"

    if git log --format=%s "$parent_ref..$b" | grep -q "^$prefix($pkg):"; then
      echo "== $prefix $pkg already committed on $b"
    elif [ "$selected" = 1 ]; then
      run_package "$pkg" "$tickets" "$name" "$(owned_keys "$pkg")"
      check_committed "$pkg" "$parent_ref..$b"
    else
      echo "== $prefix $pkg is not committed and not selected; stopping"; break
    fi
    if [ "$make_pr" = 1 ]; then open_pr "$pkg" "$b" "$parent" "$position"; fi
    parent="$b"
  done < <(rows)
  echo "done"
}

# restack: rebase each open branch onto the branch below it (or onto the base, once everything below is
# merged), with the old tip of that lower branch as the upstream, so only the branch's own commits move.
restack_run() {
  git fetch --quiet origin
  local start_branch new_parent_ref new_parent_name upstream b tip pr state pr_base
  start_branch=$(git branch --show-current)
  new_parent_ref="origin/$base"; new_parent_name="$base"; upstream="origin/$base"
  for pkg in $(chain); do
    b=$(stack_branch "$pkg")
    ensure_local "$b" || { echo "== $b does not exist; the stack ends below it"; break; }
    tip=$(git rev-parse "refs/heads/$b")
    pr=$(pr_json "$b")
    state=$( [ -n "$pr" ] && jq -r .state <<<"$pr" || echo none)
    if [ "$state" = MERGED ]; then
      pr_base=$(jq -r .baseRefName <<<"$pr")
      [ "$pr_base" = "$base" ] || die "$b was merged into $pr_base, not $base; merge the stack bottom-up into $base"
      echo "== $b is merged"
      upstream="$tip"
      continue
    fi
    if [ "$state" = CLOSED ]; then die "the PR of $b is closed; reopen it or take $pkg out of the stack by hand"; fi
    if ! git merge-base --is-ancestor "$new_parent_ref" "$b"; then
      echo "== rebasing $b onto $new_parent_name"
      if ! git rebase --quiet --onto "$new_parent_ref" "$upstream" "$b"; then
        git rebase --abort || true
        git switch --quiet "$start_branch" 2>/dev/null || true
        die "conflict while rebasing $b onto $new_parent_name; resolve it by hand (git rebase --onto $new_parent_ref $upstream $b), push it (git push --force-with-lease origin $b), then run --restack again"
      fi
      if git show-ref --verify --quiet "refs/remotes/origin/$b"; then
        git push --quiet --force-with-lease="$b:$(git rev-parse "refs/remotes/origin/$b")" origin "$b"
      fi
    fi
    if [ "$state" = OPEN ] && [ "$(jq -r .baseRefName <<<"$pr")" != "$new_parent_name" ]; then
      gh pr edit "$b" --base "$new_parent_name" >/dev/null
      echo "   $b now targets $new_parent_name"
    fi
    upstream="$tip"; new_parent_ref="$b"; new_parent_name="$b"
  done
  git switch --quiet "$start_branch" 2>/dev/null || true
  echo "restacked"
}

if [ "$restack" = 1 ]; then
  restack_run
elif [ "$stacked" = 1 ]; then
  stacked_run
else
  single_pr
fi
