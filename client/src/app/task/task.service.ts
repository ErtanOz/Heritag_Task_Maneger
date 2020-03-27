import { Injectable, EventEmitter } from '@angular/core';
import { Task } from './task.material';

@Injectable({
  providedIn: 'root'
})
export class TaskService {
  public tasks: Task[] = [];
  public selectedTaskChanged: EventEmitter<Task> = new EventEmitter();

  private selectedTaskId: string;

  constructor() {
    this.tasks.push(new Task("t0", 40, 100));
    this.tasks.push(new Task("t1", 100, 100));
    this.tasks.push(new Task("t2", 10, 100));
    this.tasks.push(new Task("t3", 10, 100));
    this.tasks.push(new Task("t4", 10, 100));
  }

  public selectTask(id: string) {
    this.selectedTaskId = id;
    this.selectedTaskChanged.emit(this.getSelectedTask());
  }

  public getSelectedTask(): Task {
    return this.getTask(this.selectedTaskId);
  }

  private getTask(id: string): Task {
    return this.tasks.find(t => t.id === id);
  }

  public getTasks(ids: string[]): Task[] {
    return this.tasks.filter(t => ids.includes(t.id));
  }

  public setProcessPoints(id: string, newProcessPoints: number) {
    this.getTask(id).processPoints = newProcessPoints;
    this.selectedTaskChanged.emit(this.getTask(id));
  }
}