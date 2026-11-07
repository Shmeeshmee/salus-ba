//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1 (docs/attachments/SPEC.md §3 O8), not a security control
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The generated video's size: large enough that its three ranges come from different parts of the stored object.
const videoSize = 50_000_000

// TestAttachments_ThePatientAttachesFilesAndThePractitionerOpensThem: the practitioner publishes a questionnaire
// with a photo-and-document item and a video item and makes it their role's default; a patient books and uploads a
// generated JPEG and a generated 50 MB MP4, which become clean, and an EICAR PDF, which is rejected and detached;
// the patient submits; the practitioner reads the QuestionnaireResponse over FHIR with both valueAttachments,
// downloads the photo whole and the video by three Range requests whose bytes match the original, and the audit
// holds an event per request. Then the web journey (test/e2e/web/attachments.spec.ts) opens the photo in the
// viewer and plays and seeks the video.
func TestAttachments_ThePatientAttachesFilesAndThePractitionerOpensThem(t *testing.T) {
	clinic, q := newClinicWithFileItems(t)
	p := newPatientWithProfile(t)
	appt := p.bookWithQuestionnaire(clinic, q.title, freeSlots(t, p.api, clinic.scheduleID)[0])
	r := p.waitForRequests(1)[0]

	photoBytes, videoBytes := photo(t), video(videoSize)
	img := p.upload(r.ID, "a-files", "Placeholder photo "+suffix(t)+".jpg", photoBytes)
	mov := p.upload(r.ID, "a-video", "Placeholder video "+suffix(t)+".mp4", videoBytes)
	bad := p.upload(r.ID, "a-files", "Placeholder document "+suffix(t)+".pdf", eicarPDF())
	for _, f := range []*uploaded{img, mov, bad} {
		if f.Status != "quarantined" {
			t.Fatalf("an upload answered status %q, want quarantined until its scan", f.Status)
		}
	}
	p.waitForStatus(img, "clean")
	p.waitForStatus(mov, "clean")
	p.waitForStatus(bad, "rejected")
	if status, _, _ := p.content(bad.ID, ""); status != http.StatusNotFound {
		t.Errorf("the rejected file's download = %d, want 404", status)
	}
	bad.reads++
	if got := attachedIDs(p.requests()[0].Item); got != img.ID+" "+mov.ID {
		t.Errorf("the patient's response holds the files %q, want the photo and the video only (the rejected file detached)", got)
	}

	p.mustWith(ifMatch(mustAtoi(t, r.Meta.VersionID)), http.StatusOK, http.MethodPost, "/questionnaire-responses/"+r.ID+"/submit", nil, nil)

	got := practitionerSearch(t, clinic, appt)
	if len(got.responses) != 1 || got.responses[0].Status != "completed" {
		t.Fatalf("the practitioner finds %d responses, want the one completed", len(got.responses))
	}
	checkAttachment(t, got.responses[0].Item, "a-files", img, "image/jpeg", photoBytes)
	checkAttachment(t, got.responses[0].Item, "a-video", mov, "video/mp4", videoBytes)

	// The practitioner's downloads: the photo whole, the video by a range at its start, its middle and its end.
	clinic.practitioner.checkWhole(img, "image/jpeg", photoBytes)
	for _, rg := range [][2]int{{0, 1 << 20}, {videoSize/2 - 1000, videoSize/2 + 1<<20}, {videoSize - 1<<20, videoSize - 1}} {
		clinic.practitioner.checkRange(mov, videoBytes, rg[0], rg[1])
	}

	patientAccount, practitionerAccount := p.me().ID, clinic.practitioner.me().ID
	checkAudit(t, img, patientAccount, practitionerAccount, "U", 1)
	checkAudit(t, mov, patientAccount, practitionerAccount, "U", 3)
	checkAudit(t, bad, patientAccount, practitionerAccount, "D", 0)

	runAttachmentsSpec(t, clinic, appt, img, mov)
}

// TestAttachments_ACancelledAppointmentTakesItsFilesWithIt: the patient uploads a file to the questionnaire of an
// appointment more than a day ahead, the file becomes clean and opens; the patient cancels; once the discard job
// has deleted the response, the file's row is gone with it (the same statement, through the foreign key) and its
// status and download are 404.
func TestAttachments_ACancelledAppointmentTakesItsFilesWithIt(t *testing.T) {
	clinic, q := newClinicWithFileItems(t)
	p := newPatientWithProfile(t)
	appt := p.bookWithQuestionnaire(clinic, q.title, slotAfter(t, p.api, clinic.scheduleID, 25*time.Hour))
	r := p.waitForRequests(1)[0]

	f := p.upload(r.ID, "a-files", "Placeholder photo "+suffix(t)+".jpg", photo(t))
	p.waitForStatus(f, "clean")
	if status, _, _ := p.content(f.ID, ""); status != http.StatusOK {
		t.Fatalf("the file's download before the cancellation = %d, want 200", status)
	}

	p.must(http.StatusOK, http.MethodPost, "/appointments/"+appt+"/cancel", nil, nil)
	checkAppointmentMail(t, p.addr, cancelledSubject, clinic.name)
	eventually(t, "the questionnaire_responses_discard job deletes the cancelled appointment's response", func() bool {
		return len(p.requests()) == 0
	})
	var rows int
	if err := ownerConn(t).QueryRow(t.Context(), `SELECT count(*) FROM "binary" WHERE id = $1`, f.ID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Errorf("the file's row outlived its response")
	}
	if got := p.do(http.MethodGet, "/binaries/"+f.ID, nil, nil); got != http.StatusNotFound {
		t.Errorf("the file's status after the cancellation = %d, want 404", got)
	}
	if status, _, _ := p.content(f.ID, ""); status != http.StatusNotFound {
		t.Errorf("the file's download after the cancellation = %d, want 404", status)
	}
}

// newClinicWithFileItems makes a bookable clinic whose practitioner publishes a questionnaire with a file upload
// item for photos and documents (up to 3 files) and one for a video, placeholder wording, and makes it their
// role's default.
func newClinicWithFileItems(t *testing.T) (*bookableClinic, questionnaireVersion) {
	t.Helper()
	clinic := newBookableClinic(t)
	q := publish(t, clinic, "Placeholder files questionnaire "+suffix(t), []map[string]any{
		{"linkId": "a-files", "text": "Placeholder photo or document question", "type": "attachment", "accept": []string{"photo", "document"}, "maxFiles": 3},
		{"linkId": "a-video", "text": "Placeholder video question", "type": "attachment", "accept": []string{"video"}, "maxFiles": 1},
	})
	clinic.practitioner.must(http.StatusOK, http.MethodPut, "/practitioner-roles/"+publicRole(t, clinic.id)+"/questionnaire",
		map[string]string{"questionnaireId": q.id}, nil)
	return clinic, q
}

// uploaded is a file as the upload answered it (the Binary schema), with the number of status and content reads
// the patient made of it, which the audit must hold.
type uploaded struct {
	ID, ContentType, Title, Status string
	Size                           int64
	reads                          int
}

// upload sends body as the app does: raw, its length declared, the title in Content-Disposition's filename*.
func (p *patient) upload(responseID, linkID, title string, body []byte) *uploaded {
	p.t.Helper()
	path := "/questionnaire-responses/" + responseID + "/items/" + linkID + "/attachments"
	req, err := http.NewRequestWithContext(p.t.Context(), http.MethodPost, p.base+path, bytes.NewReader(body))
	if err != nil {
		p.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+p.bearer)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(title))
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		p.t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	var f uploaded
	if resp.StatusCode != http.StatusCreated {
		p.t.Fatalf("POST %s = %d, want 201", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		p.t.Fatal(err)
	}
	if f.Title != title || f.Size != int64(len(body)) {
		p.t.Fatalf("the upload answered size %d and a title set %t, want %d and the title sent", f.Size, f.Title != "", len(body))
	}
	return &f
}

// waitForStatus reads the file's status, as the app polls it, until it is want: the scan is a job.
func (p *patient) waitForStatus(f *uploaded, want string) {
	p.t.Helper()
	eventually(p.t, "the binary_scan job makes the file "+want, func() bool {
		var got struct{ Status string }
		p.must(http.StatusOK, http.MethodGet, "/binaries/"+f.ID, nil, &got)
		f.reads++
		return got.Status == want
	})
}

// content downloads the file, one range when rg is not empty, with a client that waits for 50 MB.
func (a *api) content(id, rg string) (int, http.Header, []byte) {
	a.t.Helper()
	req, err := http.NewRequestWithContext(a.t.Context(), http.MethodGet, a.base+"/binaries/"+id+"/content", nil)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+a.bearer)
	if rg != "" {
		req.Header.Set("Range", rg)
	}
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		a.t.Fatalf("GET /binaries/%s/content: %v", id, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, body
}

// checkWhole downloads the whole file and checks its bytes and the headers that keep a browser from sniffing,
// caching or embedding it elsewhere.
func (a *api) checkWhole(f *uploaded, contentType string, want []byte) {
	a.t.Helper()
	status, h, body := a.content(f.ID, "")
	if status != http.StatusOK || !bytes.Equal(body, want) {
		a.t.Fatalf("the whole download of the %s = %d with %d bytes, want 200 with the %d bytes uploaded", contentType, status, len(body), len(want))
	}
	for k, v := range map[string]string{
		"Content-Type": contentType, "Accept-Ranges": "bytes", "Cache-Control": "private, no-store",
		"X-Content-Type-Options": "nosniff", "Cross-Origin-Resource-Policy": "same-origin", "ETag": `"` + sha1Hex(want) + `"`,
	} {
		if got := h.Get(k); got != v {
			a.t.Errorf("the download's %s is %q, want %q", k, got, v)
		}
	}
}

// checkRange downloads bytes from to to (inclusive) and checks they are the original's.
func (a *api) checkRange(f *uploaded, original []byte, from, to int) {
	a.t.Helper()
	status, h, body := a.content(f.ID, fmt.Sprintf("bytes=%d-%d", from, to))
	wantRange := fmt.Sprintf("bytes %d-%d/%d", from, to, len(original))
	if status != http.StatusPartialContent || h.Get("Content-Range") != wantRange || !bytes.Equal(body, original[from:to+1]) {
		a.t.Errorf("the range %d-%d = %d %q with %d bytes, want 206 %q with the original's bytes", from, to, status, h.Get("Content-Range"), len(body), wantRange)
	}
}

// checkAttachment checks the item holds exactly the file as its valueAttachment, with the absolute download URL.
func checkAttachment(t *testing.T, items []responseItem, linkID string, f *uploaded, contentType string, original []byte) {
	t.Helper()
	sum := sha1.Sum(original) //nolint:gosec // R4's Attachment.hash
	want := map[string]any{
		"contentType": contentType, "title": f.Title, "size": float64(len(original)),
		"hash": base64.StdEncoding.EncodeToString(sum[:]), "url": "/binaries/" + f.ID + "/content",
	}
	for _, it := range items {
		if it.LinkID != linkID {
			continue
		}
		if len(it.Answer) != 1 {
			t.Fatalf("the item %s holds %d answers, want the one file", linkID, len(it.Answer))
		}
		got, _ := it.Answer[0]["valueAttachment"].(map[string]any)
		for k, v := range want {
			if k == "url" {
				u, _ := got[k].(string)
				if parsed, err := url.Parse(u); err != nil || !parsed.IsAbs() || !strings.HasSuffix(u, v.(string)) {
					t.Errorf("the item %s's url is %q, want an absolute URL ending in %s", linkID, u, v)
				}
				continue
			}
			if got[k] != v {
				t.Errorf("the item %s's %s is %v, want %v", linkID, k, got[k], v)
			}
		}
		return
	}
	t.Fatalf("the completed response has no item %s", linkID)
}

// attachedIDs lists the binary ids of every valueAttachment of the items, in order.
func attachedIDs(items []responseItem) string {
	var ids []string
	for _, it := range items {
		for _, a := range it.Answer {
			if v, ok := a["valueAttachment"].(map[string]any); ok {
				u, _ := v["url"].(string)
				ids = append(ids, strings.TrimSuffix(u[strings.LastIndex(u, "/binaries/")+len("/binaries/"):], "/content"))
			}
		}
	}
	return strings.Join(ids, " ")
}

// checkAudit checks the file's events on Binary/<id>: its upload by the patient, a read for each of the patient's
// status and content requests, the scan's result (an update when clean, a delete when rejected) by binary_scan,
// and a read for each of the practitioner's downloads, ranges included.
func checkAudit(t *testing.T, f *uploaded, patientAccount, practitionerAccount, scan string, practitionerReads int) {
	t.Helper()
	var created, patientReads, scanned, staffReads int
	err := ownerConn(t).QueryRow(t.Context(), `SELECT
			count(*) FILTER (WHERE action = 'C' AND agent_account_id = $2),
			count(*) FILTER (WHERE action = 'R' AND agent_account_id = $2),
			count(*) FILTER (WHERE action = $4 AND agent_name = 'binary_scan'),
			count(*) FILTER (WHERE action = 'R' AND agent_account_id = $3)
		FROM audit_event WHERE entity_reference = 'Binary/' || $1`, f.ID, patientAccount, practitionerAccount, scan).
		Scan(&created, &patientReads, &scanned, &staffReads)
	if err != nil {
		t.Fatal(err)
	}
	if created != 1 || patientReads != f.reads || scanned != 1 || staffReads != practitionerReads {
		t.Errorf("the audit of the %s file: %d uploads, %d patient reads, %d scan results %s, %d practitioner reads; want 1, %d, 1, %d",
			f.ContentType, created, patientReads, scanned, scan, staffReads, f.reads, practitionerReads)
	}
}

// runAttachmentsSpec runs the web journey: the practitioner opens the submitted answers, the photo in the viewer,
// and plays and seeks the video.
func runAttachmentsSpec(t *testing.T, clinic *bookableClinic, appointmentID string, photo, video *uploaded) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "attachments.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
		"E2E_APPOINTMENT_ID="+appointmentID,
		"E2E_PHOTO_TITLE="+photo.Title,
		"E2E_VIDEO_TITLE="+video.Title,
		"E2E_VIDEO_ID="+video.ID,
		fmt.Sprintf("E2E_VIDEO_SECONDS=%d", videoSeconds),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the web journey failed: %v\nPlaywright:\n%s", err, out)
	}
}

func sha1Hex(b []byte) string {
	sum := sha1.Sum(b) //nolint:gosec // the download's ETag is the file's SHA-1
	return hex.EncodeToString(sum[:])
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("%q is not a number: %v", s, err)
	}
	return n
}
