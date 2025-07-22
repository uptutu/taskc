import React, { useEffect, useState } from 'react';
import {
  Table,
  Card,
  Tag,
  Button,
  Space,
  Select,
  DatePicker,
  Typography,
  Row,
  Col,
  Statistic,
  Alert as AntAlert
} from 'antd';
import {
  ReloadOutlined,
  BellOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { taskApi } from '@/api';
import { Alert, AlertLevel } from '@/types';

const { Title } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

const AlertList: React.FC = () => {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [filters, setFilters] = useState({
    level: '',
    dateRange: null as [dayjs.Dayjs, dayjs.Dayjs] | null,
  });

  useEffect(() => {
    loadAlerts();
  }, [pagination.current, pagination.pageSize, filters.level]);

  const loadAlerts = async () => {
    try {
      setLoading(true);
      const response = await taskApi.getAlerts(
        pagination.current,
        pagination.pageSize,
        filters.level
      );
      
      setAlerts(response.data || []);
      setPagination(prev => ({
        ...prev,
        total: response.total || 0,
      }));
    } catch (error) {
      console.error('Failed to load alerts:', error);
    } finally {
      setLoading(false);
    }
  };

  const getLevelColor = (level: AlertLevel) => {
    switch (level) {
      case AlertLevel.CRITICAL:
        return 'red';
      case AlertLevel.WARNING:
        return 'orange';
      case AlertLevel.INFO:
        return 'blue';
      default:
        return 'default';
    }
  };

  const getLevelIcon = (level: AlertLevel) => {
    switch (level) {
      case AlertLevel.CRITICAL:
        return <CloseCircleOutlined style={{ color: '#f5222d' }} />;
      case AlertLevel.WARNING:
        return <ExclamationCircleOutlined style={{ color: '#fa8c16' }} />;
      case AlertLevel.INFO:
        return <InfoCircleOutlined style={{ color: '#1890ff' }} />;
      default:
        return <BellOutlined />;
    }
  };

  const columns = [
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: AlertLevel) => (
        <Space>
          {getLevelIcon(level)}
          <Tag color={getLevelColor(level)}>
            {level}
          </Tag>
        </Space>
      ),
      filters: [
        { text: '严重', value: AlertLevel.CRITICAL },
        { text: '警告', value: AlertLevel.WARNING },
        { text: '信息', value: AlertLevel.INFO },
      ],
      onFilter: (value: string, record: Alert) => record.level === value,
    },
    {
      title: '任务ID',
      dataIndex: 'task_id',
      key: 'task_id',
      width: 120,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
      width: 300,
    },
    {
      title: '通知渠道',
      dataIndex: 'channels',
      key: 'channels',
      width: 120,
      render: (channels: string) => {
        try {
          const channelList = JSON.parse(channels);
          return (
            <Space direction="vertical" size="small">
              {channelList.map((channel: string) => (
                <Tag key={channel} size="small">
                  {channel.toUpperCase()}
                </Tag>
              ))}
            </Space>
          );
        } catch {
          return channels;
        }
      },
    },
    {
      title: '状态',
      dataIndex: 'sent',
      key: 'sent',
      width: 80,
      render: (sent: boolean) => (
        <Tag color={sent ? 'green' : 'orange'}>
          {sent ? '已发送' : '待发送'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => new Date(text).toLocaleString(),
      sorter: (a: Alert, b: Alert) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    {
      title: '发送时间',
      dataIndex: 'sent_at',
      key: 'sent_at',
      width: 180,
      render: (text: string | null) => text ? new Date(text).toLocaleString() : '-',
    },
  ];

  const alertStats = React.useMemo(() => {
    const critical = alerts.filter(a => a.level === AlertLevel.CRITICAL).length;
    const warning = alerts.filter(a => a.level === AlertLevel.WARNING).length;
    const info = alerts.filter(a => a.level === AlertLevel.INFO).length;
    const sent = alerts.filter(a => a.sent).length;
    
    return { critical, warning, info, sent, total: alerts.length };
  }, [alerts]);

  return (
    <div>
      <Title level={2}>告警中心</Title>
      
      {/* 统计信息 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card>
            <Statistic
              title="总告警数"
              value={alertStats.total}
              prefix={<BellOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="严重告警"
              value={alertStats.critical}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="警告告警"
              value={alertStats.warning}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="信息告警"
              value={alertStats.info}
              prefix={<InfoCircleOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="已发送"
              value={alertStats.sent}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="待发送"
              value={alertStats.total - alertStats.sent}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 严重告警提示 */}
      {alertStats.critical > 0 && (
        <AntAlert
          message={`当前有 ${alertStats.critical} 个严重告警需要处理`}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Select
              placeholder="筛选告警级别"
              style={{ width: 140 }}
              allowClear
              value={filters.level || undefined}
              onChange={(value) => setFilters(prev => ({ ...prev, level: value || '' }))}
            >
              <Option value={AlertLevel.CRITICAL}>严重</Option>
              <Option value={AlertLevel.WARNING}>警告</Option>
              <Option value={AlertLevel.INFO}>信息</Option>
            </Select>
            
            <RangePicker
              placeholder={['开始日期', '结束日期']}
              value={filters.dateRange}
              onChange={(dates) => setFilters(prev => ({ ...prev, dateRange: dates }))}
            />
          </Space>
          
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={loadAlerts}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={alerts}
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
          scroll={{ x: 1200 }}
          rowClassName={(record) => {
            if (record.level === AlertLevel.CRITICAL) return 'alert-critical-row';
            if (record.level === AlertLevel.WARNING) return 'alert-warning-row';
            return '';
          }}
        />
      </Card>

      <style>
        {`
          .alert-critical-row {
            background-color: #fff2f0 !important;
          }
          .alert-warning-row {
            background-color: #fff7e6 !important;
          }
        `}
      </style>
    </div>
  );
};

export default AlertList;