import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Card,
  Row,
  Col,
  Descriptions,
  Tag,
  Button,
  Space,
  Tabs,
  Table,
  Typography,
  Statistic,
  Alert,
  Spin
} from 'antd';
import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { taskApi } from '@/api';
import { Task, TaskStatus, TaskHealth, TaskMetrics, Alert as AlertType } from '@/types';

const { Title, Paragraph } = Typography;
const { TabPane } = Tabs;

const TaskDetail: React.FC = () => {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const [task, setTask] = useState<Task | null>(null);
  const [health, setHealth] = useState<TaskHealth | null>(null);
  const [metrics, setMetrics] = useState<TaskMetrics | null>(null);
  const [alerts, setAlerts] = useState<AlertType[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (taskId) {
      loadTaskData();
    }
  }, [taskId]);

  const loadTaskData = async () => {
    if (!taskId) return;

    try {
      setLoading(true);
      const [taskResponse, healthResponse, metricsResponse, alertsResponse] = await Promise.all([
        taskApi.getTask(taskId),
        taskApi.getTaskHealth(taskId),
        taskApi.getTaskMetrics(taskId, 24),
        taskApi.getTaskAlerts(taskId, 20)
      ]);

      setTask(taskResponse);
      setHealth(healthResponse);
      setMetrics(metricsResponse);
      setAlerts(alertsResponse);
    } catch (error) {
      console.error('Failed to load task data:', error);
    } finally {
      setLoading(false);
    }
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

  const getMetricsChartOption = () => {
    if (!metrics) return {};

    const hours = Array.from({ length: 24 }, (_, i) => {
      const time = new Date();
      time.setHours(time.getHours() - 23 + i);
      return time.getHours() + ':00';
    });

    // 模拟心跳数据
    const heartbeatData = Array.from({ length: 24 }, () => Math.floor(Math.random() * 60) + 10);

    return {
      title: {
        text: '24小时心跳趋势',
        textStyle: { fontSize: 16 }
      },
      tooltip: {
        trigger: 'axis',
        formatter: '{b}: {c} 次心跳'
      },
      xAxis: {
        type: 'category',
        data: hours
      },
      yAxis: {
        type: 'value',
        name: '心跳次数'
      },
      series: [
        {
          name: '心跳次数',
          type: 'line',
          data: heartbeatData,
          smooth: true,
          areaStyle: {
            opacity: 0.3
          },
          itemStyle: {
            color: '#1890ff'
          }
        }
      ]
    };
  };

  const alertColumns = [
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (level: string) => {
        const color = level === 'CRITICAL' ? 'red' : level === 'WARNING' ? 'orange' : 'blue';
        return <Tag color={color}>{level}</Tag>;
      }
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => new Date(text).toLocaleString(),
    },
  ];

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!task) {
    return (
      <Alert
        message="任务不存在"
        description="请检查任务ID是否正确"
        type="error"
        action={
          <Button onClick={() => navigate('/tasks')}>
            返回任务列表
          </Button>
        }
      />
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/tasks')}
          >
            返回
          </Button>
          <Title level={2} style={{ margin: 0 }}>
            任务详情: {task.name}
          </Title>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        {/* 基本信息 */}
        <Col span={24}>
          <Card 
            title="基本信息"
            extra={
              <Button
                icon={<ReloadOutlined />}
                onClick={loadTaskData}
                loading={loading}
              >
                刷新
              </Button>
            }
          >
            <Descriptions column={3}>
              <Descriptions.Item label="任务ID">{task.task_id}</Descriptions.Item>
              <Descriptions.Item label="任务名称">{task.name}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Space>
                  {getStatusIcon(task.status)}
                  <Tag color={getStatusColor(task.status)}>
                    {task.status}
                  </Tag>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={3}>
                <Paragraph>{task.description || '无描述'}</Paragraph>
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(task.created_at).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {new Date(task.updated_at).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="最后心跳">
                {health?.last_heartbeat 
                  ? new Date(health.last_heartbeat).toLocaleString()
                  : '无心跳记录'
                }
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* 统计信息 */}
        <Col span={24}>
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="24小时心跳数"
                  value={metrics?.heartbeat_count || 0}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="24小时告警数"
                  value={metrics?.alert_count || 0}
                  valueStyle={{ color: '#f5222d' }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="运行时间比例"
                  value={(metrics?.uptime_ratio || 0) * 100}
                  precision={2}
                  suffix="%"
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="平均CPU使用率"
                  value={health?.resource_usage?.avg_cpu || 0}
                  precision={1}
                  suffix="%"
                  valueStyle={{ color: '#fa8c16' }}
                />
              </Card>
            </Col>
          </Row>
        </Col>

        {/* 详细信息标签页 */}
        <Col span={24}>
          <Card>
            <Tabs defaultActiveKey="metrics">
              <TabPane tab="监控指标" key="metrics">
                <ReactECharts 
                  option={getMetricsChartOption()} 
                  style={{ height: '400px' }} 
                />
              </TabPane>
              
              <TabPane tab="告警历史" key="alerts">
                <Table
                  columns={alertColumns}
                  dataSource={alerts}
                  rowKey="id"
                  pagination={{
                    pageSize: 10,
                    showSizeChanger: false,
                  }}
                  size="small"
                />
              </TabPane>
              
              <TabPane tab="探测历史" key="probes">
                <Table
                  columns={[
                    { title: '时间', dataIndex: 'timestamp', render: (text: string) => new Date(text).toLocaleString() },
                    { title: '结果', dataIndex: 'result', render: (text: string) => <Tag color={text === 'SUCCESS' ? 'green' : 'red'}>{text}</Tag> },
                    { title: '延迟(ms)', dataIndex: 'latency_ms' },
                    { title: '错误信息', dataIndex: 'error_message', ellipsis: true },
                  ]}
                  dataSource={health?.probe_history || []}
                  rowKey="timestamp"
                  pagination={false}
                  size="small"
                />
              </TabPane>
              
              <TabPane tab="资源使用" key="resources">
                <Row gutter={16}>
                  <Col span={12}>
                    <Card title="CPU使用率">
                      <Statistic
                        value={health?.resource_usage?.avg_cpu || 0}
                        precision={1}
                        suffix="%"
                        valueStyle={{ 
                          color: (health?.resource_usage?.avg_cpu || 0) > 80 ? '#f5222d' : '#52c41a' 
                        }}
                      />
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card title="内存使用">
                      <Statistic
                        value={health?.resource_usage?.max_mem_mb || 0}
                        suffix="MB"
                        valueStyle={{ color: '#1890ff' }}
                      />
                    </Card>
                  </Col>
                </Row>
              </TabPane>
            </Tabs>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default TaskDetail;