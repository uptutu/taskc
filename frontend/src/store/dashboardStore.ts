import { create } from 'zustand';

interface DashboardState {
  stats: {
    totalTasks: number;
    healthyTasks: number;
    suspectedTasks: number;
    failedTasks: number;
    totalAlerts: number;
  };
  realtimeUpdates: boolean;
  
  // Actions
  setStats: (stats: DashboardState['stats']) => void;
  updateTaskCount: (status: string, delta: number) => void;
  setRealtimeUpdates: (enabled: boolean) => void;
}

export const useDashboardStore = create<DashboardState>((set) => ({
  stats: {
    totalTasks: 0,
    healthyTasks: 0,
    suspectedTasks: 0,
    failedTasks: 0,
    totalAlerts: 0,
  },
  realtimeUpdates: true,

  setStats: (stats) => set({ stats }),

  updateTaskCount: (status, delta) => set((state) => {
    const newStats = { ...state.stats };
    switch (status) {
      case 'HEALTHY':
        newStats.healthyTasks += delta;
        break;
      case 'SUSPECTED':
        newStats.suspectedTasks += delta;
        break;
      case 'FAILED':
        newStats.failedTasks += delta;
        break;
    }
    newStats.totalTasks += delta;
    return { stats: newStats };
  }),

  setRealtimeUpdates: (enabled) => set({ realtimeUpdates: enabled }),
}));