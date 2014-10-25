package que

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func openTestClientConninfo(conninfo string) (*Client, error) {
	datname := os.Getenv("PGDATABASE")
	sslmode := os.Getenv("PGSSLMODE")
	timeout := os.Getenv("PGCONNECT_TIMEOUT")

	if datname == "" {
		os.Setenv("PGDATABASE", "que-go-test")
	}

	if sslmode == "" {
		os.Setenv("PGSSLMODE", "disable")
	}

	if timeout == "" {
		os.Setenv("PGCONNECT_TIMEOUT", "20")
	}

	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		return nil, err
	}
	return &Client{db: db}, nil
}

func openTestClient(t testing.TB) *Client {
	c, err := openTestClientConninfo("")
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func truncateAndClose(db *sql.DB) {
	if _, err := db.Exec("TRUNCATE TABLE que_jobs"); err != nil {
		panic(err)
	}
	db.Close()
}

func TestEnqueueEmpty(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	if err := c.Enqueue(Job{Type: "MyJob"}); err != nil {
		t.Fatal(err)
	}

	j, err := findOneJob(c.db)
	if err != nil {
		t.Fatal(err)
	}

	// check resulting job
	if want := 100; j.Priority != want {
		t.Errorf("want Priority=%d, got %d", want, j.Priority)
	}
	if j.RunAt.IsZero() {
		t.Error("want non-zero RunAt")
	}
	if j.ID == 0 {
		t.Errorf("want non-zero ID")
	}
	if want := "MyJob"; j.Type != want {
		t.Errorf("want Type=%q, got %q", want, j.Type)
	}
	if want := "[]"; j.Args != want {
		t.Errorf("want Args=%s, got %s", want, j.Args)
	}
	if want := 0; j.ErrorCount != want {
		t.Errorf("want ErrorCount=%d, got %d", want, j.ErrorCount)
	}
	if want := 100; j.Priority != want {
		t.Errorf("want priority=%d, got %d", want, j.Priority)
	}
	if want := ""; j.Queue != want {
		t.Errorf("want Queue=%q, got %q", want, j.Queue)
	}

}

func TestEnqueueWithPriority(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	want := 99
	if err := c.Enqueue(Job{Type: "MyJob", Priority: want}); err != nil {
		t.Fatal(err)
	}

	j, err := findOneJob(c.db)
	if err != nil {
		t.Fatal(err)
	}

	if j.Priority != want {
		t.Errorf("want Priority=%d, got %d", want, j.Priority)
	}
}

func TestEnqueueWithRunAt(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	want := time.Now().Add(2 * time.Minute)
	if err := c.Enqueue(Job{Type: "MyJob", RunAt: want}); err != nil {
		t.Fatal(err)
	}

	j, err := findOneJob(c.db)
	if err != nil {
		t.Fatal(err)
	}

	// round to the microsecond as postgres does
	want = want.Round(time.Microsecond)
	if !want.Equal(j.RunAt) {
		t.Errorf("want RunAt=%s, got %s", want, j.RunAt)
	}
}

func TestEnqueueWithArgs(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	want := `{"arg1":0, "arg2":"a string"}`
	if err := c.Enqueue(Job{Type: "MyJob", Args: want}); err != nil {
		t.Fatal(err)
	}

	j, err := findOneJob(c.db)
	if err != nil {
		t.Fatal(err)
	}

	if j.Args != want {
		t.Errorf("want Args=%s, got %s", want, j.Args)
	}
}

func TestEnqueueWithQueue(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	want := "special-work-queue"
	if err := c.Enqueue(Job{Type: "MyJob", Queue: want}); err != nil {
		t.Fatal(err)
	}

	j, err := findOneJob(c.db)
	if err != nil {
		t.Fatal(err)
	}

	if j.Queue != want {
		t.Errorf("want Queue=%q, got %q", want, j.Queue)
	}
}

// TODO: test Enqueue with empty Type
func TestEnqueueWithEmptyType(t *testing.T) {
	c := openTestClient(t)
	defer truncateAndClose(c.db)

	if err := c.Enqueue(Job{Type: ""}); err != ErrMissingType {
		t.Fatalf("want ErrMissingType, got %v", err)
	}
}

func findOneJob(db *sql.DB) (*Job, error) {
	findSQL := `
	SELECT priority, run_at, job_id, job_class, args, error_count, last_error, queue
	FROM que_jobs LIMIT 1`

	j := &Job{}
	err := db.QueryRow(findSQL).Scan(
		&j.Priority,
		&j.RunAt,
		&j.ID,
		&j.Type,
		&j.Args,
		&j.ErrorCount,
		&j.LastError,
		&j.Queue,
	)
	return j, err
}