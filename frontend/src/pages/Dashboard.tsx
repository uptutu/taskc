import React, { useEffect, useState, useCallback } from 'react';
import { Row, Col, Card, Statistic, Progress, Table, Tag, Typography, Space } from 'antd';
import { 
  AppstoreOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { taskApi } from '@/api';
import { useDashboardStore } from '@/store/dashboardStore';
import { Task, TaskStatus } from '@/types';

const { Title } = Typography;

const Dashboard: React.FC = () => {
  const { stats, setStats } = useDashboardStore();
  const [recentTasks, setRecentTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      
      // 加载任务统计
      const tasksResponse = await taskApi.getTasks(1, 100);
      const tasks = tasksResponse.data || [];
      
      const taskStats = {
        totalTasks: tasks.length,
        healthyTasks: tasks.filter(t => t.status === TaskStatus.HEALTHY).length,
        suspectedTasks: tasks.filter(t => t.status === TaskStatus.SUSPECTED).length,
        failedTasks: tasks.filter(t => t.status === TaskStatus.FAILED).length,
        totalAlerts: 0, // 这里应该从告警API获取
      };
      
      setStats(taskStats);
      setRecentTasks(tasks.slice(0, 10));
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setLoading(false);
    }
  }, [setStats]);

  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  const getStatusOption = () => {
    return {
      title: {
        text: '任务状态分布',
        left: 'center',
        textStyle: {
          fontSize: 16,
        }
      },
      tooltip: {
        trigger: 'item',
        formatter: '{a} <br/>{b}: {c} ({d}%)'
      },
      series: [
        {
          name: '任务状态',
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['50%', '60%'],
          data: [
            { value: stats.healthyTasks, name: '健康', itemStyle: { color: '#52c41a' } },
            { value: stats.suspectedTasks, name: '疑似', itemStyle: { color: '#fa8c16' } },
            { value: stats.failedTasks, name: '失败', itemStyle: { color: '#f5222d' } },
          ],
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
          }
        }
      ]
    };
  };

  const getTrendOption = () => {
    const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`);
    const healthyData = Array.from({ length: 24 }, () => Math.floor(Math.random() * 100));
    const failedData = Array.from({ length: 24 }, () => Math.floor(Math.random() * 20));

    return {
      title: {
        text: '24小时趋势',
        left: 'center',
        textStyle: {
          fontSize: 16,
        }
      },
      tooltip: {
        trigger: 'axis'
      },
      legend: {
        top: '10%',
        data: ['健康任务', '失败任务']
      },
      xAxis: {
        type: 'category',
        data: hours
      },
      yAxis: {
        type: 'value'
      },
      series: [
        {
          name: '健康任务',
          type: 'line',
          data: healthyData,
          smooth: true,
          itemStyle: { color: '#52c41a' }
        },
        {
          name: '失败任务',
          type: 'line',
          data: failedData,
          smooth: true,
          itemStyle: { color: '#f5222d' }
        }
      ]
    };
  };

  const getStatusColor = (status: TaskStatus) => {
    switch (status) {
      case TaskStatus.HEALTHY:
        return 'green';
      case TaskStatus.SUSPECTED:
        return 'orange';
      case TaskStatus.FAILED:
        return 'red';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: TaskStatus) => {
    switch (status) {
      case TaskStatus.HEALTHY:
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case TaskStatus.SUSPECTED:
        return <ExclamationCircleOutlined style={{ color: '#fa8c16' }} />;
      case TaskStatus.FAILED:
        return <CloseCircleOutlined style={{ color: '#f5222d' }} />;
      default:
        return null;
    }
  };

  const taskColumns = [
    {
      title: '任务ID',
      dataIndex: 'task_id',
      key: 'task_id',
      width: 120,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: TaskStatus) => (
        <Space>
          {getStatusIcon(status)}
          <Tag color={getStatusColor(status)}>
            {status}
          </Tag>
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      render: (text: string) => new Date(text).toLocaleString(),
    },
  ];

  const healthyRate = stats.totalTasks > 0 ? (stats.healthyTasks / stats.totalTasks) * 100 : 0;

  return (
    <div>
      <Title level={2}>系统仪表板</Title>
      
      <Row gutter={[16, 16]}>
        {/* 统计卡片 */}
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="总任务数"
              value={stats.totalTasks}
              prefix={<AppstoreOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="健康任务"
              value={stats.healthyTasks}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="疑似任务"
              value={stats.suspectedTasks}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="失败任务"
              value={stats.failedTasks}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* 健康率 */}
        <Col xs={24} md={8}>
          <Card title="系统健康率">
            <Progress
              type="dashboard"
              percent={Math.round(healthyRate)}
              strokeColor={{
                '0%': '#f5222d',
                '50%': '#fa8c16',
                '100%': '#52c41a',
              }}
              format={(percent) => `${percent}%`}
            />
          </Card>
        </Col>

        {/* 状态分布图 */}
        <Col xs={24} md={8}>
          <Card>
            <ReactECharts option={getStatusOption()} style={{ height: '300px' }} />
          </Card>
        </Col>

        {/* 趋势图 */}
        <Col xs={24} md={8}>
          <Card>
            <ReactECharts option={getTrendOption()} style={{ height: '300px' }} />
          </Card>
        </Col>
      </Row>

      {/* 最近任务 */}
      <Row style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="最近任务" extra={<a href="/tasks">查看全部</a>}>
            <Table
              columns={taskColumns}
              dataSource={recentTasks}
              rowKey="id"
              pagination={false}
              loading={loading}
              size="small"
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;