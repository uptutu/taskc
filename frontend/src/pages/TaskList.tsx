import React, { useEffect, useState } from 'react';
import { 
  Table, 
  Card, 
  Button, 
  Space, 
  Tag, 
  Modal, 
  Form, 
  Input, 
  Select, 
  message,
  Typography,
  Row,
  Col,
  Statistic
} from 'antd';
import { 
  PlusOutlined, 
  ReloadOutlined, 
  EyeOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { taskApi } from '@/api';
import { useTaskStore } from '@/store/taskStore';
import { Task, TaskStatus } from '@/types';

const { Title } = Typography;
const { Option } = Select;

const TaskList: React.FC = () => {
  const navigate = useNavigate();
  const { tasks, setTasks, addTask, removeTask, loading, setLoading } = useTaskStore();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [filters, setFilters] = useState({
    status: '',
  });
  const [form] = Form.useForm();

  useEffect(() => {
    loadTasks();
  }, [pagination.current, pagination.pageSize, filters.status]);

  const loadTasks = async () => {
    try {
      setLoading(true);
      const response = await taskApi.getTasks(
        pagination.current,
        pagination.pageSize,
        filters.status
      );
      
      setTasks(response.data || []);
      setPagination(prev => ({
        ...prev,
        total: response.total || 0,
      }));
    } catch (error) {
      console.error('Failed to load tasks:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTask = async (values: any) => {
    try {
      const newTask = await taskApi.createTask(values);
      addTask(newTask);
      setCreateModalVisible(false);
      form.resetFields();
      message.success('任务创建成功');
    } catch (error) {
      console.error('Failed to create task:', error);
    }
  };

  const handleDeleteTask = async (taskId: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个任务吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await taskApi.deleteTask(taskId);
          removeTask(taskId);
          message.success('任务删除成功');
        } catch (error) {
          console.error('Failed to delete task:', error);
        }
      },
    });
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

  const columns = [
    {
      title: '任务ID',
      dataIndex: 'task_id',
      key: 'task_id',
      width: 120,
      fixed: 'left' as const,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: TaskStatus) => (
        <Space>
          {getStatusIcon(status)}
          <Tag color={getStatusColor(status)}>
            {status}
          </Tag>
        </Space>
      ),
      filters: [
        { text: '健康', value: TaskStatus.HEALTHY },
        { text: '疑似', value: TaskStatus.SUSPECTED },
        { text: '失败', value: TaskStatus.FAILED },
      ],
      onFilter: (value: string, record: Task) => record.status === value,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => new Date(text).toLocaleString(),
      sorter: (a: Task, b: Task) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (text: string) => new Date(text).toLocaleString(),
      sorter: (a: Task, b: Task) => new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime(),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right' as const,
      render: (_, record: Task) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/tasks/${record.task_id}`)}
          >
            查看
          </Button>
          <Button
            type="link"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDeleteTask(record.task_id)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const statusStats = React.useMemo(() => {
    const healthy = tasks.filter(t => t.status === TaskStatus.HEALTHY).length;
    const suspected = tasks.filter(t => t.status === TaskStatus.SUSPECTED).length;
    const failed = tasks.filter(t => t.status === TaskStatus.FAILED).length;
    
    return { healthy, suspected, failed, total: tasks.length };
  }, [tasks]);

  return (
    <div>
      <Title level={2}>任务管理</Title>
      
      {/* 统计信息 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总任务数"
              value={statusStats.total}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="健康任务"
              value={statusStats.healthy}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="疑似任务"
              value={statusStats.suspected}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="失败任务"
              value={statusStats.failed}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Select
              placeholder="筛选状态"
              style={{ width: 120 }}
              allowClear
              value={filters.status || undefined}
              onChange={(value) => setFilters({ status: value || '' })}
            >
              <Option value={TaskStatus.HEALTHY}>健康</Option>
              <Option value={TaskStatus.SUSPECTED}>疑似</Option>
              <Option value={TaskStatus.FAILED}>失败</Option>
            </Select>
          </Space>
          
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={loadTasks}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalVisible(true)}
            >
              创建任务
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={tasks}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: (page, size) => {
              setPagination(prev => ({
                ...prev,
                current: page,
                pageSize: size || 20,
              }));
            },
          }}
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* 创建任务模态框 */}
      <Modal
        title="创建新任务"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateTask}
        >
          <Form.Item
            name="task_id"
            label="任务ID"
            rules={[
              { required: true, message: '请输入任务ID' },
              { pattern: /^[a-zA-Z0-9_-]+$/, message: '任务ID只能包含字母、数字、下划线和短横线' }
            ]}
          >
            <Input placeholder="例如: user-service-001" />
          </Form.Item>

          <Form.Item
            name="name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="例如: 用户服务" />
          </Form.Item>

          <Form.Item
            name="description"
            label="任务描述"
          >
            <Input.TextArea
              rows={3}
              placeholder="请输入任务描述"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TaskList;