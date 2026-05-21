import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { changePassword, exportSystemData } from "@/services/system";
import { checkHealth, checkReady } from "@/services/health";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { ErrorPanel } from "@/components/error-panel";
import { Download, ShieldCheck, CheckCircle2, Activity, Server, Database, KeyRound, FolderTree, AlertCircle, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { StatusBadge } from "@/components/status-badge";

export function SystemPage() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  const { data: healthStatus, refetch: refetchHealth, isFetching: healthFetching } = useQuery({
    queryKey: ["health"],
    queryFn: checkHealth,
    refetchInterval: 30000,
  });

  const { data: readyStatus, refetch: refetchReady, isFetching: readyFetching } = useQuery({
    queryKey: ["ready"],
    queryFn: checkReady,
    refetchInterval: 30000,
  });

  const passwordMutation = useMutation({
    mutationFn: changePassword,
    onSuccess: () => {
      setPasswordSuccess(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success("密码修改成功");
      setTimeout(() => setPasswordSuccess(false), 3000);
    },
    onError: (error: any) => {
      toast.error("修改密码失败", { description: error.message });
    }
  });

  const exportMutation = useMutation({
    mutationFn: exportSystemData,
    onSuccess: (blob) => {
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `vaultfleet-export-${new Date().toISOString().split("T")[0]}.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      toast.success("数据导出成功");
    },
    onError: (error: any) => {
      toast.error("数据导出失败", { description: error.message });
    }
  });

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error("新密码不匹配");
      return;
    }
    passwordMutation.mutate({ current_password: currentPassword, new_password: newPassword });
  };

  const isRefreshing = healthFetching || readyFetching;

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight">系统管理</h1>
        <Button 
          variant="outline" 
          size="sm" 
          onClick={() => { refetchHealth(); refetchReady(); }}
          disabled={isRefreshing}
        >
          <RefreshCw className={isRefreshing ? "h-4 w-4 mr-2 animate-spin" : "h-4 w-4 mr-2"} />
          刷新状态
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Activity className="h-5 w-5 text-primary" />
            系统状态
          </CardTitle>
          <CardDescription>Master 服务运行及依赖组件就绪状态。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-6 md:grid-cols-2">
          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center gap-3">
                <Server className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="text-sm font-medium">服务进程</div>
                  <div className="text-xs text-muted-foreground">HTTP API Server</div>
                </div>
              </div>
              <StatusBadge status={healthStatus?.ok ? "success" : "failed"} />
            </div>

            <div className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center gap-3">
                <Database className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="text-sm font-medium">数据库连接</div>
                  <div className="text-xs text-muted-foreground">SQLite 存储</div>
                </div>
              </div>
              <StatusBadge status={readyStatus?.ok || readyStatus?.status === "ready" ? "success" : "failed"} />
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center gap-3">
                <KeyRound className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="text-sm font-medium">Master Key</div>
                  <div className="text-xs text-muted-foreground">数据加密密钥</div>
                </div>
              </div>
              <StatusBadge status={readyStatus?.ok ? "success" : "failed"} />
            </div>

            <div className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center gap-3">
                <FolderTree className="h-5 w-5 text-muted-foreground" />
                <div>
                  <div className="text-sm font-medium">数据目录</div>
                  <div className="text-xs text-muted-foreground">本地存储可用性</div>
                </div>
              </div>
              <StatusBadge status={readyStatus?.ok ? "success" : "failed"} />
            </div>
          </div>
          
          {!readyStatus?.ok && readyStatus?.error && (
            <div className="md:col-span-2 flex items-start gap-2 text-red-600 bg-red-50 p-3 rounded border border-red-200 text-xs">
              <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
              <div>
                <span className="font-bold mr-1">系统未就绪:</span>
                {readyStatus.error}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>修改密码</CardTitle>
            <CardDescription>定期修改密码以确保账户安全。</CardDescription>
          </CardHeader>
          <form onSubmit={handlePasswordSubmit}>
            <CardContent className="space-y-4">
              <ErrorPanel error={passwordMutation.error as any} />
              {passwordSuccess && (
                <div className="flex items-center gap-2 text-green-600 text-sm font-medium bg-green-50 p-3 rounded border border-green-200">
                  <CheckCircle2 className="h-4 w-4" /> 密码修改成功
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="current">当前密码</Label>
                <Input
                  id="current"
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new">新密码</Label>
                <Input
                  id="new"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm">确认新密码</Label>
                <Input
                  id="confirm"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button type="submit" disabled={passwordMutation.isPending}>
                {passwordMutation.isPending ? "正在修改..." : "提交修改"}
              </Button>
            </CardFooter>
          </form>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>数据导出</CardTitle>
            <CardDescription>导出 Master 节点的完整数据库。建议在进行系统迁移或重大更新前导出备份。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col items-center justify-center py-8 text-center space-y-4 bg-muted/20 rounded-lg border-2 border-dashed">
              <ShieldCheck className="h-12 w-12 text-muted-foreground opacity-30" />
              <p className="text-sm text-muted-foreground px-6">
                导出的压缩包包含 SQLite 数据库文件。请务必加密存储导出的文件。
              </p>
            </div>
          </CardContent>
          <CardFooter>
            <Button 
              variant="outline" 
              className="w-full" 
              onClick={() => exportMutation.mutate()}
              disabled={exportMutation.isPending}
            >
              <Download className="mr-2 h-4 w-4" /> 
              {exportMutation.isPending ? "正在生成导出文件..." : "导出 Master 数据"}
            </Button>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
