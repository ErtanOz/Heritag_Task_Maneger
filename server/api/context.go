package api

import (
	"database/sql"
	"github.com/hauke96/simple-task-manager/server/auth"
	"github.com/hauke96/simple-task-manager/server/database"
	"github.com/hauke96/simple-task-manager/server/permission"
	"github.com/hauke96/simple-task-manager/server/project"
	"github.com/hauke96/simple-task-manager/server/task"
	"github.com/hauke96/simple-task-manager/server/util"
	"github.com/pkg/errors"
)

type Context struct {
	util.Logger
	Token          *auth.Token
	Transaction    *sql.Tx
	ProjectService *project.ProjectService
	TaskService    *task.TaskService
}

// createContext starts a new Transaction and creates new service instances which use this new Transaction so that all
// services (also those calling each other) are using the same Transaction.
func createContext(token *auth.Token) (*Context, error) {
	ctx := &Context{}
	ctx.Token = token

	tx, err := database.GetTransaction()
	if err != nil {
		return nil, errors.Wrap(err, "error getting Transaction")
	}
	ctx.Transaction = tx

	ctx.LogTraceId = util.GetLogTraceId()

	permissionService := permission.Init(tx, ctx.LogTraceId)
	ctx.TaskService = task.Init(tx, ctx.LogTraceId, permissionService)
	ctx.ProjectService = project.Init(tx, ctx.LogTraceId, ctx.TaskService, permissionService)

	return ctx, nil
}