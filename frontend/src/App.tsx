import React, { Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import Layout from '@/components/Layout';
import 'antd/dist/reset.css';

// Lazy load pages
const Dashboard = React.lazy(() => import('@/pages/Dashboard'));
const TaskList = React.lazy(() => import('@/pages/TaskList'));
const TaskDetail = React.lazy(() => import('@/pages/TaskDetail'));
const AlertList = React.lazy(() => import('@/pages/AlertList'));

const App: React.FC = () => {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1890ff',
          borderRadius: 6,
        },
      }}
    >
      <Router>
        <Layout>
          <Suspense fallback={
            <div style={{ 
              display: 'flex', 
              justifyContent: 'center', 
              alignItems: 'center', 
              height: '50vh' 
            }}>
              <Spin size="large" />
            </div>
          }>
            <Routes>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/tasks" element={<TaskList />} />
              <Route path="/tasks/:taskId" element={<TaskDetail />} />
              <Route path="/alerts" element={<AlertList />} />
            </Routes>
          </Suspense>
        </Layout>
      </Router>
    </ConfigProvider>
  );
};

export default App;