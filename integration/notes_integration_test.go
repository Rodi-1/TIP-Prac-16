package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"example.com/pz16/internal/db"
	"example.com/pz16/internal/httpapi"
	"example.com/pz16/internal/models"
	"example.com/pz16/internal/repo"
	"example.com/pz16/internal/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func init() { gin.SetMode(gin.TestMode) }

var schemaCounter atomic.Int64

type environment struct {
	db  *sql.DB
	srv *httptest.Server
}

func setup(t *testing.T) environment {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("set DB_DSN and run docker compose up -d --wait")
	}
	admin, e := db.Open(dsn)
	require.NoError(t, e)
	schema := fmt.Sprintf("pz16_%d_%d", time.Now().UnixNano(), schemaCounter.Add(1))
	_, e = admin.Exec(`CREATE SCHEMA "` + schema + `"`)
	require.NoError(t, e)
	t.Cleanup(func() {
		_, e := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
		assert.NoError(t, e)
		admin.Close()
	})
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	d, e := db.Open(u.String())
	require.NoError(t, e)
	t.Cleanup(func() { d.Close() })
	require.NoError(t, db.ApplyMigrations(d))
	r := gin.New()
	r.Use(gin.Recovery())
	s := service.Service{Notes: repo.NoteRepo{DB: d}}
	httpapi.Router{Svc: &s}.Register(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return environment{db: d, srv: srv}
}
func request(t *testing.T, e environment, method, path, body string, want int) ([]byte, http.Header) {
	t.Helper()
	q, err := http.NewRequest(method, e.srv.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	q.Header.Set("Content-Type", "application/json")
	response, err := e.srv.Client().Do(q)
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, want, response.StatusCode, string(data))
	if want != 204 {
		assert.Contains(t, response.Header.Get("Content-Type"), "application/json")
	}
	t.Logf("%s %s -> %d", method, path, response.StatusCode)
	return data, response.Header
}
func create(t *testing.T, e environment, body string) models.Note {
	t.Helper()
	data, _ := request(t, e, "POST", "/notes", body, 201)
	var n models.Note
	require.NoError(t, json.Unmarshal(data, &n))
	require.Positive(t, n.ID)
	require.False(t, n.CreatedAt.IsZero())
	return n
}
func TestCRUD(t *testing.T) {
	t.Parallel()
	e := setup(t)
	fixture, err := os.ReadFile("testdata/create.json")
	require.NoError(t, err)
	n := create(t, e, string(fixture))
	var title, content string
	require.NoError(t, e.db.QueryRow(`SELECT title,content FROM notes WHERE id=$1`, n.ID).Scan(&title, &content))
	assert.Equal(t, "Hello", title)
	assert.Equal(t, "World", content)
	data, _ := request(t, e, "GET", fmt.Sprintf("/notes/%d", n.ID), "", 200)
	var got models.Note
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, n.ID, got.ID)
	assert.True(t, n.CreatedAt.Equal(got.CreatedAt))
	time.Sleep(2 * time.Millisecond)
	data, _ = request(t, e, "PATCH", fmt.Sprintf("/notes/%d", n.ID), `{"content":"Changed"}`, 200)
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "Hello", got.Title)
	assert.Equal(t, "Changed", got.Content)
	assert.True(t, got.CreatedAt.Equal(n.CreatedAt))
	assert.True(t, got.UpdatedAt.After(n.UpdatedAt))
	var updated time.Time
	require.NoError(t, e.db.QueryRow(`SELECT content,updated_at FROM notes WHERE id=$1`, n.ID).Scan(&content, &updated))
	assert.Equal(t, "Changed", content)
	assert.True(t, updated.Equal(got.UpdatedAt))
	data, _ = request(t, e, "DELETE", fmt.Sprintf("/notes/%d", n.ID), "", 204)
	assert.Empty(t, data)
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM notes WHERE id=$1`, n.ID).Scan(&count))
	assert.Zero(t, count)
	request(t, e, "GET", fmt.Sprintf("/notes/%d", n.ID), "", 404)
}
func TestValidationAndMissing(t *testing.T) {
	t.Parallel()
	e := setup(t)
	for _, c := range []struct {
		name, method, path, body string
		want                     int
	}{{"json", "POST", "/notes", "{", 400}, {"empty", "POST", "/notes", `{"title":" "}`, 400}, {"wrong type", "POST", "/notes", `{"title":1}`, 400}, {"bad id", "GET", "/notes/abc", "", 400}, {"zero id", "GET", "/notes/0", "", 400}, {"missing", "GET", "/notes/999", "", 404}, {"patch empty", "PATCH", "/notes/999", `{"title":" "}`, 400}, {"patch missing", "PATCH", "/notes/999", `{"title":"New"}`, 404}, {"delete missing", "DELETE", "/notes/999", "", 404}, {"limit", "GET", "/notes?limit=0", "", 400}, {"page", "GET", "/notes?page=-1", "", 400}} {
		t.Run(c.name, func(t *testing.T) { request(t, e, c.method, c.path, c.body, c.want) })
	}
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&count))
	assert.Zero(t, count)
}
func TestPaginationAndFilter(t *testing.T) {
	t.Parallel()
	e := setup(t)
	for _, title := range []string{"Alpha", "Beta", "Alphabet"} {
		create(t, e, fmt.Sprintf(`{"title":%q,"content":"x"}`, title))
	}
	data, headers := request(t, e, "GET", "/notes?q=alpha&limit=1&page=2", "", 200)
	var items []models.Note
	require.NoError(t, json.Unmarshal(data, &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Alphabet", items[0].Title)
	assert.Equal(t, "2", headers.Get("X-Total-Count"))
	data, _ = request(t, e, "GET", "/notes?q=alpha&limit=1&page=3", "", 200)
	assert.JSONEq(t, `[]`, string(data))
}
func TestConcurrentCreates(t *testing.T) {
	t.Parallel()
	e := setup(t)
	var wg sync.WaitGroup
	ids := make(chan int64, 10)
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			response, err := http.Post(e.srv.URL+"/notes", "application/json", strings.NewReader(fmt.Sprintf(`{"title":"Concurrent %d","content":"x"}`, i)))
			if err != nil {
				errs <- err
				return
			}
			defer response.Body.Close()
			if response.StatusCode != 201 {
				errs <- fmt.Errorf("status %d", response.StatusCode)
				return
			}
			var n models.Note
			if err = json.NewDecoder(response.Body).Decode(&n); err != nil {
				errs <- err
				return
			}
			ids <- n.ID
		}(i)
	}
	wg.Wait()
	close(errs)
	close(ids)
	for err := range errs {
		require.NoError(t, err)
	}
	unique := make(map[int64]bool)
	for id := range ids {
		require.False(t, unique[id])
		unique[id] = true
	}
	assert.Len(t, unique, 10)
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&count))
	assert.Equal(t, 10, count)
}
func TestTransactionsAndMigrations(t *testing.T) {
	t.Parallel()
	e := setup(t)
	require.NoError(t, db.ApplyMigrations(e.db))
	var indexes int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename='notes'`).Scan(&indexes))
	assert.GreaterOrEqual(t, indexes, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO notes(title,content) VALUES($1,$2)`, "Must rollback", "x")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO notes(title,content) VALUES($1,$2)`, "", "x")
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&count))
	assert.Zero(t, count)
	tx, err = e.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO notes(title,content) VALUES($1,$2),($3,$4)`, "First", "1", "Second", "2")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&count))
	assert.Equal(t, 2, count)
}
