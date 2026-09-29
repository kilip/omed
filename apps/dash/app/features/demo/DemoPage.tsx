import {
  ArrowUpOutlined,
  BellOutlined,
  DollarCircleOutlined,
  PlusOutlined,
  SearchOutlined,
  UserOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Avatar,
  Badge,
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Divider,
  Drawer,
  Dropdown,
  Flex,
  Form,
  Input,
  InputNumber,
  Modal,
  message,
  Progress,
  Radio,
  Row,
  Segmented,
  Select,
  Slider,
  Statistic,
  Steps,
  Switch,
  Table,
  type TableProps,
  Tabs,
  Tag,
  Timeline,
  Tooltip,
  Typography,
} from "antd";
import { type PropsWithChildren, useState } from "react";
import { DashboardLayout } from "~/shared/ui/DashboardLayout";

const { Title, Text, Paragraph } = Typography;

export const meta = [{ title: "UI Showcase" }];

interface Row_ {
  key: string;
  code: string;
  name: string;
  type: "asset" | "liability" | "equity" | "revenue" | "expense";
  balance: number;
  status: "active" | "archived";
}

const rows: Row_[] = [
  {
    key: "1",
    code: "1-1000",
    name: "Kas & Bank",
    type: "asset",
    balance: 125_400_000,
    status: "active",
  },
  {
    key: "2",
    code: "2-1000",
    name: "Hutang Usaha",
    type: "liability",
    balance: 32_000_000,
    status: "active",
  },
  {
    key: "3",
    code: "3-1000",
    name: "Modal Disetor",
    type: "equity",
    balance: 200_000_000,
    status: "active",
  },
  {
    key: "4",
    code: "4-1000",
    name: "Pendapatan Jasa",
    type: "revenue",
    balance: 88_750_000,
    status: "active",
  },
  {
    key: "5",
    code: "5-1000",
    name: "Beban Server",
    type: "expense",
    balance: 6_200_000,
    status: "archived",
  },
];

const typeColor: Record<Row_["type"], string> = {
  asset: "blue",
  liability: "volcano",
  equity: "purple",
  revenue: "green",
  expense: "gold",
};

const idr = (n: number) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);

const columns: TableProps<Row_>["columns"] = [
  { title: "Kode", dataIndex: "code", render: (v) => <Text code>{v}</Text> },
  { title: "Nama", dataIndex: "name" },
  {
    title: "Tipe",
    dataIndex: "type",
    render: (t: Row_["type"]) => <Tag color={typeColor[t]}>{t}</Tag>,
  },
  {
    title: "Saldo",
    dataIndex: "balance",
    align: "right",
    render: (v) => idr(v),
  },
  {
    title: "Status",
    dataIndex: "status",
    render: (s: Row_["status"]) => (
      <Badge status={s === "active" ? "success" : "default"} text={s} />
    ),
  },
];

function Section({ title, children }: PropsWithChildren<{ title: string }>) {
  return (
    <section style={{ marginBottom: 32 }}>
      <Title level={4} style={{ marginBottom: 16 }}>
        {title}
      </Title>
      {children}
    </section>
  );
}

export default function DemoPage() {
  const [modalOpen, setModalOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [msg, ctx] = message.useMessage();
  const [form] = Form.useForm();

  return (
    <div>
      {ctx}
      <Title level={2} style={{ marginTop: 0 }}>
        UI Showcase
      </Title>
      <Paragraph type="secondary">
        Semua komponen yang dipakai di Omed, dengan tema glassmorphism. Toggle
        light/dark di header.
      </Paragraph>
      <Divider />

      <Section title="Typography & Buttons">
        <Flex vertical gap={16}>
          <div>
            <Title level={1} style={{ margin: 0 }}>
              Heading 1
            </Title>
            <Title level={3} style={{ margin: 0 }}>
              Heading 3
            </Title>
            <Text>Teks biasa, </Text>
            <Text type="secondary">secondary, </Text>
            <Text type="success">success, </Text>
            <Text type="danger">danger, </Text>
            <Text code>code</Text>
          </div>
          <Flex gap={12} wrap>
            <Button type="primary">Primary</Button>
            <Button>Default</Button>
            <Button type="dashed">Dashed</Button>
            <Button type="text">Text</Button>
            <Button type="link">Link</Button>
            <Button danger>Danger</Button>
            <Button type="primary" icon={<PlusOutlined />}>
              Dengan Icon
            </Button>
            <Button type="primary" loading>
              Loading
            </Button>
            <Button shape="circle" icon={<SearchOutlined />} />
            <Button disabled>Disabled</Button>
          </Flex>
        </Flex>
      </Section>

      <Section title="Stat Cards">
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12} xl={6}>
            <Card>
              <Statistic title="Total Aset" value={125400000} prefix="Rp" />
            </Card>
          </Col>
          <Col xs={24} md={12} xl={6}>
            <Card>
              <Statistic
                title="Pendapatan"
                value={12.5}
                precision={1}
                suffix="%"
                prefix={<ArrowUpOutlined />}
                styles={{ content: { color: "#22c55e" } }}
              />
            </Card>
          </Col>
          <Col xs={24} md={12} xl={6}>
            <Card>
              <Statistic
                title="Jurnal Bulan Ini"
                value={342}
                prefix={<DollarCircleOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} md={12} xl={6}>
            <Card>
              <Text type="secondary">Periode Terbuka</Text>
              <Progress percent={68} style={{ marginTop: 8 }} />
            </Card>
          </Col>
        </Row>
      </Section>

      <Section title="Form">
        <Card>
          <Form
            form={form}
            layout="vertical"
            initialValues={{
              currency: "IDR",
              type: "asset",
              active: true,
              level: 30,
            }}
            onFinish={() => msg.success("Form tersubmit")}
          >
            <Row gutter={16}>
              <Col xs={24} md={8}>
                <Form.Item
                  label="Kode Akun"
                  name="code"
                  rules={[{ required: true }]}
                >
                  <Input placeholder="1-1000" />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item
                  label="Nama Akun"
                  name="name"
                  rules={[{ required: true }]}
                >
                  <Input prefix={<UserOutlined />} placeholder="Kas & Bank" />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Tipe" name="type">
                  <Select
                    options={[
                      "asset",
                      "liability",
                      "equity",
                      "revenue",
                      "expense",
                    ].map((v) => ({ value: v, label: v }))}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Mata Uang" name="currency">
                  <Radio.Group
                    optionType="button"
                    buttonStyle="solid"
                    options={["IDR", "USD", "EUR"]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Saldo Awal" name="balance">
                  <InputNumber style={{ width: "100%" }} prefix="Rp" min={0} />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Tanggal" name="date">
                  <DatePicker style={{ width: "100%" }} />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Level" name="level">
                  <Slider />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item label="Aktif" name="active" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item name="agree" valuePropName="checked" label=" ">
                  <Checkbox>Setuju syarat & ketentuan</Checkbox>
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item label="Deskripsi" name="description">
                  <Input.TextArea rows={3} />
                </Form.Item>
              </Col>
            </Row>
            <Flex gap={12} justify="flex-end">
              <Button onClick={() => form.resetFields()}>Reset</Button>
              <Button type="primary" htmlType="submit">
                Simpan
              </Button>
            </Flex>
          </Form>
        </Card>
      </Section>

      <Section title="Table">
        <Card styles={{ body: { padding: 0 } }}>
          <Table columns={columns} dataSource={rows} pagination={false} />
        </Card>
      </Section>

      <Section title="Tabs, Segmented & Steps">
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card>
              <Tabs
                items={[
                  {
                    key: "1",
                    label: "Ringkasan",
                    children: "Konten ringkasan.",
                  },
                  { key: "2", label: "Detail", children: "Konten detail." },
                  { key: "3", label: "Riwayat", children: "Konten riwayat." },
                ]}
              />
              <Segmented options={["Harian", "Mingguan", "Bulanan"]} block />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card>
              <Steps
                current={1}
                items={[
                  { title: "Draft" },
                  { title: "Closed" },
                  { title: "Locked" },
                ]}
                style={{ marginBottom: 24 }}
              />
              <Timeline
                items={[
                  { content: "Periode dibuat" },
                  { content: "Jurnal diposting", color: "green" },
                  { content: "Menunggu penutupan", color: "gray" },
                ]}
              />
            </Card>
          </Col>
        </Row>
      </Section>

      <Section title="Feedback">
        <Flex vertical gap={12}>
          <Alert
            type="success"
            showIcon
            title="Berhasil"
            description="Entry berhasil diposting."
          />
          <Alert
            type="info"
            showIcon
            title="Info"
            description="Periode akan ditutup akhir bulan."
          />
          <Alert
            type="warning"
            showIcon
            title="Peringatan"
            description="Ada entry yang belum seimbang."
          />
          <Alert
            type="error"
            showIcon
            title="Error"
            description="Tidak bisa posting ke akun archived."
          />
          <Flex gap={12} wrap align="center">
            <Button onClick={() => msg.info("Halo dari message")}>
              Message
            </Button>
            <Button onClick={() => setModalOpen(true)}>Modal</Button>
            <Button onClick={() => setDrawerOpen(true)}>Drawer</Button>
            <Tooltip title="Ini tooltip">
              <Button>Hover aku</Button>
            </Tooltip>
            <Dropdown
              menu={{
                items: [
                  { key: "1", label: "Edit" },
                  { key: "2", label: "Arsipkan" },
                  { key: "3", danger: true, label: "Hapus" },
                ],
              }}
            >
              <Button>Dropdown</Button>
            </Dropdown>
            <Badge count={5}>
              <Avatar shape="square" icon={<BellOutlined />} />
            </Badge>
            <Avatar.Group>
              <Avatar style={{ background: "#6366f1" }}>T</Avatar>
              <Avatar style={{ background: "#ec4899" }}>O</Avatar>
              <Avatar style={{ background: "#14b8a6" }}>M</Avatar>
            </Avatar.Group>
          </Flex>
        </Flex>
      </Section>

      <Modal
        title="Contoh Modal"
        open={modalOpen}
        onOk={() => setModalOpen(false)}
        onCancel={() => setModalOpen(false)}
      >
        <Paragraph>Modal ikut efek kaca dari tema.</Paragraph>
      </Modal>
      <Drawer
        title="Contoh Drawer"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        size={420}
      >
        <Paragraph>Drawer juga ikut efek kaca.</Paragraph>
      </Drawer>
    </div>
  );
}
