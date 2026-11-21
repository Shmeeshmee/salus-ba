-- The clinic's membership (docs/plans/clinic-cms.md, package D). Step 0 holds the first query; package D adds the rest.

-- name: ActivePlans :many
SELECT code, name, price_per_seat_minor, currency, features, active
FROM subscription_plan
WHERE active
ORDER BY code;
