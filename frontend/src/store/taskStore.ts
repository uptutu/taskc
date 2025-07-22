import { create } from 'zustand';
import { Task, TaskStatus } from '@/types';

interface TaskState {
  tasks: Task[];
  selectedTask: Task | null;
  loading: boolean;
  error: string | null;
  
  // Actions
  setTasks: (tasks: Task[]) => void;
  setSelectedTask: (task: Task | null) => void;
  addTask: (task: Task) => void;
  updateTask: (taskId: string, updates: Partial<Task>) => void;
  removeTask: (taskId: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  selectedTask: null,
  loading: false,
  error: null,

  setTasks: (tasks) => set({ tasks }),

  setSelectedTask: (task) => set({ selectedTask: task }),

  addTask: (task) => set((state) => ({
    tasks: [task, ...state.tasks]
  })),

  updateTask: (taskId, updates) => set((state) => ({
    tasks: state.tasks.map(task =>
      task.task_id === taskId ? { ...task, ...updates } : task
    ),
    selectedTask: state.selectedTask?.task_id === taskId
      ? { ...state.selectedTask, ...updates }
      : state.selectedTask
  })),

  removeTask: (taskId) => set((state) => ({
    tasks: state.tasks.filter(task => task.task_id !== taskId),
    selectedTask: state.selectedTask?.task_id === taskId ? null : state.selectedTask
  })),

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error }),
}));