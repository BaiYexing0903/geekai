package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetAdminIdReadsAdminUserId(t *testing.T) {
	ctx := &gin.Context{}
	ctx.Set("ADMIN_USER_ID", 1)

	base := BaseHandler{}
	if got := base.GetAdminId(ctx); got != 1 {
		t.Fatalf("expected admin id 1, got %d", got)
	}
}
