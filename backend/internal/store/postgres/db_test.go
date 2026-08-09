package postgres

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/deepcut/live/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testutil.SetupDB(ctx)
	if err != nil {
		log.Fatalf("setup test db: %v", err)
	}
	testPool = pool
	defer cleanup()

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// mockTx implements pgx.Tx for unit tests that only need an interface
// satisfier. Methods that are not called by the code under test return
// zero values or sentinel errors.
// ---------------------------------------------------------------------------

type mockTx struct{}

func (mockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (mockTx) Commit(ctx context.Context) error          { return errors.New("not implemented") }
func (mockTx) Rollback(ctx context.Context) error        { return errors.New("not implemented") }
func (mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (mockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (mockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}
func (mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}
func (mockTx) Conn() *pgx.Conn { return nil }

var _ pgx.Tx = mockTx{}

// ---------------------------------------------------------------------------
// TestNew — verify New returns a non-nil *Queries with the db set.
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	q := New(mockTx{})
	if q == nil {
		t.Fatal("New() returned nil")
	}
	if q.db == nil {
		t.Fatal("db field is nil")
	}
}

// ---------------------------------------------------------------------------
// TestWithTx — verify WithTx returns a new *Queries backed by the given tx.
// ---------------------------------------------------------------------------

func TestWithTx(t *testing.T) {
	original := New(mockTx{})
	tx := mockTx{}

	qt := original.WithTx(tx)

	if qt == nil {
		t.Fatal("WithTx() returned nil")
	}
	if qt == original {
		t.Fatal("WithTx() returned the same *Queries pointer, want a new one")
	}
	if qt.db == nil {
		t.Fatal("db field is nil")
	}
	// The db field should be the tx we passed in.
	if qt.db != tx {
		t.Fatal("db field is not the tx passed to WithTx")
	}
}

// ---------------------------------------------------------------------------
// TestGetUser — integration test against a real Postgres container.
// ---------------------------------------------------------------------------

func TestGetUser(t *testing.T) {
	testutil.SkipOnShort(t)
	if err := testutil.TruncateAll(context.Background(), testPool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	ctx := context.Background()
	q := New(testPool)

	// Seed a user so we have a real UUID to query.
	var seedID pgtype.UUID
	err := testPool.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, stream_key_hash)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		"g-test", "test@test.com", "Test User", "hash-abc",
	).Scan(&seedID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// UUID that does not exist in the database.
	nonexistentUUID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	tests := []struct {
		name       string
		id         pgtype.UUID
		wantErr    bool
		errIs      error // compare with errors.Is
		errContain string
		check      func(t *testing.T, u User)
	}{
		{
			name:    "happy path",
			id:      seedID,
			wantErr: false,
			check: func(t *testing.T, u User) {
				if u.GoogleID != "g-test" {
					t.Fatalf("GoogleID = %q, want %q", u.GoogleID, "g-test")
				}
				if u.Email != "test@test.com" {
					t.Fatalf("Email = %q, want %q", u.Email, "test@test.com")
				}
				if u.Name != "Test User" {
					t.Fatalf("Name = %q, want %q", u.Name, "Test User")
				}
				if u.StreamKeyHash != "hash-abc" {
					t.Fatalf("StreamKeyHash = %q, want %q", u.StreamKeyHash, "hash-abc")
				}
				if u.IsLive {
					t.Fatal("IsLive = true, want false for newly created user")
				}
			},
		},
		{
			name:       "not found",
			id:         nonexistentUUID,
			wantErr:    true,
			errIs:      pgx.ErrNoRows,
			errContain: "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := q.GetUser(ctx, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Fatalf("expected error to be %v, got %v", tt.errIs, err)
				}
				if tt.errContain != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContain)) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContain, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, u)
			}
		})
	}
}
