package httpapi

import (
	"context"
	"errors"
	"example.com/pz16/internal/models"
	"example.com/pz16/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Router struct{ Svc *service.Service }

func (rt Router) Register(r *gin.Engine) {
	r.POST("/notes", rt.create)
	r.GET("/notes", rt.list)
	r.GET("/notes/:id", rt.get)
	r.PATCH("/notes/:id", rt.patch)
	r.DELETE("/notes/:id", rt.del)
}
func reqCtx(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), 5*time.Second)
}
func id(c *gin.Context) (int64, bool) {
	n, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || n <= 0 {
		c.JSON(400, gin.H{"error": "invalid_id"})
		return 0, false
	}
	return n, true
}
func failure(c *gin.Context, e error) {
	switch {
	case errors.Is(e, models.ErrNotFound):
		c.JSON(404, gin.H{"error": "not_found"})
	case errors.Is(e, models.ErrInvalid):
		c.JSON(400, gin.H{"error": "invalid_note"})
	default:
		log.Printf("database error: %v", e)
		c.JSON(500, gin.H{"error": "database_error"})
	}
}
func (rt Router) create(c *gin.Context) {
	var in models.NoteCreate
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid_json"})
		return
	}
	n := models.Note{Title: in.Title, Content: in.Content}
	ctx, cancel := reqCtx(c)
	defer cancel()
	if e := rt.Svc.Create(ctx, &n); e != nil {
		failure(c, e)
		return
	}
	c.JSON(http.StatusCreated, n)
}
func (rt Router) get(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	ctx, cancel := reqCtx(c)
	defer cancel()
	n, e := rt.Svc.Get(ctx, id)
	if e != nil {
		failure(c, e)
		return
	}
	c.JSON(200, n)
}
func (rt Router) list(c *gin.Context) {
	limit, page := 20, 1
	var e error
	if s := c.Query("limit"); s != "" {
		limit, e = strconv.Atoi(s)
		if e != nil || limit < 1 || limit > 100 {
			c.JSON(400, gin.H{"error": "invalid_limit"})
			return
		}
	}
	if s := c.Query("page"); s != "" {
		page, e = strconv.Atoi(s)
		if e != nil || page < 1 || page > 1000000 {
			c.JSON(400, gin.H{"error": "invalid_page"})
			return
		}
	}
	ctx, cancel := reqCtx(c)
	defer cancel()
	items, total, e := rt.Svc.List(ctx, c.Query("q"), limit, (page-1)*limit)
	if e != nil {
		failure(c, e)
		return
	}
	c.Header("X-Total-Count", strconv.Itoa(total))
	c.JSON(200, items)
}
func (rt Router) patch(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	var in models.NoteUpdate
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid_json"})
		return
	}
	ctx, cancel := reqCtx(c)
	defer cancel()
	n, e := rt.Svc.Patch(ctx, id, in)
	if e != nil {
		failure(c, e)
		return
	}
	c.JSON(200, n)
}
func (rt Router) del(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	ctx, cancel := reqCtx(c)
	defer cancel()
	if e := rt.Svc.Delete(ctx, id); e != nil {
		failure(c, e)
		return
	}
	c.Status(204)
}
