package project

import (
	"database/sql"
	"fmt"
	"github.com/hauke96/simple-task-manager/server/permission"
	"github.com/hauke96/simple-task-manager/server/task"
	"github.com/hauke96/simple-task-manager/server/util"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"strconv"
)

// Helper struct to read raw data from database. The "Project" struct has higher-level structure (e.g. arrays), which we
// don't have in the database columns.
type projectRow struct {
	id          int
	name        string
	taskIds     []string
	users       []string
	owner       string
	description string
}

type storePg struct {
	db    *sql.DB
	table string
}

func (s *storePg) init(db *sql.DB) {
	s.db = db
	s.table = "projects"
}

func (s *storePg) getProjects(userId string) ([]*Project, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE $1=ANY(users)", s.table)

	util.LogQuery(query, userId)

	rows, err := s.db.Query(query, userId)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	projects := make([]*Project, 0)
	for rows.Next() {
		project, err := rowToProject(rows)
		if err != nil {
			return nil, errors.Wrap(err, "error converting row into project")
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (s *storePg) getProject(projectId string) (*Project, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id=$1", s.table)
	return execQuery(s.db, query, projectId)
}

func (s *storePg) getProjectByTask(taskId string) (*Project, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE $1=ANY(task_ids)", s.table)
	return execQuery(s.db, query, taskId)
}

// areTasksUsed checks whether any of the given tasks is already part of a project. Returns false and an error in case
// of an error.
func (s *storePg) areTasksUsed(taskIds []string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE task_ids && $1", s.table)

	util.LogQuery(query, taskIds)
	rows, err := s.db.Query(query, pq.Array(taskIds))
	if err != nil {
		return false, errors.Wrap(err, "could not run query")
	}
	defer rows.Close()

	ok := rows.Next()
	if !ok {
		return false, errors.New("there is no next row or an error happened")
	}

	var count int
	err = rows.Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "could not scan count from rows")
	}

	return count != 0, nil
}

// Adds the given project draft and assigns an ID to the project
func (s *storePg) addProject(draft *Project) (*Project, error) {
	query := fmt.Sprintf("INSERT INTO %s (name, task_ids, description, users, owner) VALUES($1, $2, $3, $4, $5) RETURNING *", s.table)

	return execQuery(s.db, query, draft.Name, pq.Array(draft.TaskIDs), draft.Description, pq.Array(draft.Users), draft.Owner)
}

func (s *storePg) addUser(projectId string, userIdToAdd string) (*Project, error) {
	originalProject, err := s.getProject(projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting project with ID '%s'", projectId)
	}

	newUsers := append(originalProject.Users, userIdToAdd)

	query := fmt.Sprintf("UPDATE %s SET users=$1 WHERE id=$2 RETURNING *", s.table)
	return execQuery(s.db, query, pq.Array(newUsers), projectId)
}

func (s *storePg) removeUser(projectId string, userIdToRemove string) (*Project, error) {
	originalProject, err := s.getProject(projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting project with ID '%s'", projectId)
	}

	remainingUsers := make([]string, 0)
	for _, u := range originalProject.Users {
		if u != userIdToRemove {
			remainingUsers = append(remainingUsers, u)
		}
	}

	query := fmt.Sprintf("UPDATE %s SET users=$1 WHERE id=$2 RETURNING *", s.table)
	return execQuery(s.db, query, pq.Array(remainingUsers), projectId)
}

func (s *storePg) delete(projectId string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id=$1", s.table)

	_, err := s.db.Exec(query, projectId)
	return err
}

// getTasks will get the tasks for the given projectId and also checks the ownership of the given user.
func (s *storePg) getTasks(projectId string, userId string) ([]*task.Task, error) {
	p, err := s.getProject(projectId)
	if err != nil {
		return nil, err
	}

	return task.GetTasks(p.TaskIDs, userId)
}
func (s *storePg) updateName(projectId string, newName string) (*Project, error) {
	query := fmt.Sprintf("UPDATE %s SET name=$1 WHERE id=$2 RETURNING *", s.table)
	return execQuery(s.db, query, newName, projectId)
}

func (s *storePg) updateDescription(projectId string, newDescription string) (*Project, error) {
	query := fmt.Sprintf("UPDATE %s SET description=$1 WHERE id=$2 RETURNING *", s.table)
	return execQuery(s.db, query, newDescription, projectId)
}

// execQuery executed the given query, turns the result into a Project object and closes the query.
func execQuery(db *sql.DB, query string, params ...interface{}) (*Project, error) {
	util.LogQuery(query, params...)
	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, errors.Wrap(err, "could not run query")
	}
	defer rows.Close()

	ok := rows.Next()
	if !ok {
		return nil, errors.New("there is no next row or an error happened")
	}

	p, err := rowToProject(rows)

	if p == nil && err == nil {
		return nil, errors.New(fmt.Sprintf("Project does not exist"))
	}

	return p, err
}

// rowToProject turns the current row into a Project object. This does not close the row.
func rowToProject(rows *sql.Rows) (*Project, error) {
	var p projectRow
	err := rows.Scan(&p.id, &p.name, &p.owner, &p.description, pq.Array(&p.taskIds), pq.Array(&p.users))
	if err != nil {
		return nil, errors.Wrap(err, "could not scan rows")
	}

	result := Project{}

	result.Id = strconv.Itoa(p.id)
	result.Name = p.name
	result.Users = p.users
	result.Owner = p.owner
	result.Description = p.description
	result.TaskIDs = p.taskIds

	needsAssignment, err := permission.AssignmentInProjectNeeded(result.Id)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to get assignment requirement for newly read project %s", result.Id))
	}
	result.NeedsAssignment = needsAssignment

	return &result, nil
}
