"use client";
import { GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { signIn } from "@omed/auth/client";
import { Button, Card, Space, Typography } from "antd";
import { useRouter } from "next/navigation";

const { Title, Text } = Typography;

export default function LoginForm() {
  const router = useRouter();
  const signInWith = async (provider: string) => {
    await signIn.social({
      provider,
      fetchOptions: {
        onSuccess: () => {
          router.push("/");
        },
      },
    });
  };
  return (
    <Card
      style={{
        width: "100%",
        maxWidth: 400,
        boxShadow: "0 4px 12px rgba(0, 0, 0, 0.08)",
        borderRadius: "8px",
      }}
    >
      <div style={{ textAlign: "center", marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>
          Login to Omed
        </Title>
        <Text type="secondary">Sign in to continue to your account</Text>
      </div>
      {/* OAuth Social Login Buttons */}
      <Space orientation="vertical" style={{ width: "100%" }} size="middle">
        <Button
          block
          size="large"
          icon={<GoogleOutlined />}
          onClick={async () => await signInWith("google")}
          style={{
            borderColor: "#d9d9d9",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          Sign in with Google
        </Button>

        <Button
          block
          size="large"
          icon={<GithubOutlined />}
          onClick={async () => await signInWith("github")}
          style={{
            backgroundColor: "#24292e",
            color: "#fff",
            borderColor: "#24292e",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          Sign in with GitHub
        </Button>
      </Space>
    </Card>
  );
}
