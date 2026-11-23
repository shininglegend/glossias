# Backend checks. Integration targets need Docker running (they skip without it).

.PHONY: test vet test-integration vet-integration query-report query-baseline

test:
	go test ./...

vet:
	go vet ./...

# Real-Postgres tests (testcontainers). -count=1: the database is outside Go's test cache.
test-integration:
	go test -tags integration -count=1 -timeout 10m ./src/integration/... ./src/pkg/testdb/...

vet-integration:
	go vet -tags integration ./...

# Per-endpoint DB query counts against the committed baseline.
query-report:
	go test -tags integration -count=1 ./src/integration -run TestQueryBudgets -v

# Re-record the baseline after lowering query counts; commit the result.
query-baseline:
	go test -tags integration -count=1 ./src/integration -run TestQueryBudgets -update-baseline
