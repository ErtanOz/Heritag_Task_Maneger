package task

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/hauke96/sigolo"
	"strconv"
	"strings"
)

type taskRow struct {
	id               int
	processPoints    int
	maxProcessPoints int
	geometry         string
	assignedUser     string
}

type storePg struct {
	db    *sql.DB
	table string
}

func (s *storePg) init(db *sql.DB) {
	s.db = db
	s.table = "tasks"
}

func (s *storePg) getTasks(taskIds []string) ([]*Task, error) {
	ids := strings.Join(taskIds, ",")
	query := fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)", s.table, ids)
	sigolo.Debug(query)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	tasks := make([]*Task, 0)
	for rows.Next() {
		var t taskRow
		err = rows.Scan(&t.id, &t.processPoints, &t.maxProcessPoints, &t.geometry, &t.assignedUser)
		if err != nil {
			return nil, err
		}

		task, err := rowToTask(t)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *storePg) getTask(id string) (*Task, error) {
	tasks, err := s.getTasks([]string{id})
	if err != nil {
		return nil, err
	}

	return tasks[0], nil
}

func (s *storePg) addTasks(newTasks []*Task) ([]*Task, error) {
	taskIds := make([]string, 0)

	for _, t := range newTasks {
		id, err := s.addTask(t)
		if err != nil {
			return nil, err
		}

		taskIds = append(taskIds, id)
	}

	return s.getTasks(taskIds)
}

func (s *storePg) addTask(task *Task) (string, error) {
	geometryBytes, err := json.Marshal(task.Geometry)
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("INSERT INTO %s(process_points, max_process_points, geometry, assigned_user) VALUES('%d', '%d', '%s', '%s') RETURNING id",
		s.table, task.ProcessPoints, task.MaxProcessPoints, string(geometryBytes), task.AssignedUser)
	sigolo.Debug(query)
	row := s.db.QueryRow(query)

	var taskId int
	err = row.Scan(&taskId)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(taskId), nil
}

func (s *storePg) assignUser(id, user string) (*Task, error) {
	query := fmt.Sprintf("UPDATE %s SET assigned_user='%s' WHERE id=%s", s.table, user, id)
	sigolo.Debug(query)
	rows := s.db.QueryRow(query)

	var task taskRow
	err := rows.Scan(&task.id, &task.processPoints, &task.maxProcessPoints, &task.geometry, &task.assignedUser)
	if err != nil {
		return nil, err
	}

	result, err := rowToTask(task)
	return result, err
}

func (s *storePg) unassignUser(id, user string) (*Task, error) {
	query := fmt.Sprintf("UPDATE %s SET assigned_user=NULL WHERE id=%s", s.table, id)
	sigolo.Debug(query)
	rows := s.db.QueryRow(query)

	var task taskRow
	err := rows.Scan(&task.id, &task.processPoints, &task.maxProcessPoints, &task.geometry, &task.assignedUser)
	if err != nil {
		return nil, err
	}

	result, err := rowToTask(task)
	return result, err
}

func (s *storePg) setProcessPoints(id string, newPoints int) (*Task, error) {
	panic("implement me")
}

func rowToTask(p taskRow) (*Task, error) {
	result := Task{}

	result.Id = strconv.Itoa(p.id)
	result.ProcessPoints = p.processPoints
	result.MaxProcessPoints = p.maxProcessPoints
	result.AssignedUser = p.assignedUser

	var g [][]float64
	err := json.Unmarshal([]byte(p.geometry), &g)
	if err != nil {
		return nil, err
	}
	result.Geometry = g

	return &result, nil
}