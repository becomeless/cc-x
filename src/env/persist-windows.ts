/**
 * Windows 持久化：直写 HKCU\Environment + 单次 WM_SETTINGCHANGE 广播。
 *
 * 评审③：钉 `powershell.exe`（5.1，Windows 必然存在），不用 `pwsh`（7+，可能没装）。
 * 逻辑搬自现版 Set-UserEnv-Fast + Invoke-EnvBroadcast：注册表瞬时写入、最后只广播一次（100ms 短超时、
 * SMTO_ABORTIFHUNG 跳过挂死窗口），避免「逐个 setx 广播 9 次、每窗口等 1s」的拖慢。
 *
 * 键值通过环境变量传 JSON 给子进程、用 ConvertFrom-Json 解析 —— 彻底避开命令行注入/引号转义。
 */
import { spawnSync } from 'node:child_process';

/** key → 值（null/'' 表示删除该用户环境变量）。 */
export type EnvVals = Record<string, string | null>;

const PS_SCRIPT = String.raw`
$ErrorActionPreference = 'Stop'
$m = $env:CCX_ENV_PAYLOAD | ConvertFrom-Json
$reg = 'HKCU:\Environment'
foreach ($p in $m.PSObject.Properties) {
  if ($null -eq $p.Value -or $p.Value -eq '') {
    Remove-ItemProperty -Path $reg -Name $p.Name -ErrorAction SilentlyContinue
  } else {
    Set-ItemProperty -Path $reg -Name $p.Name -Value $p.Value -Type String
  }
}
$sig = @'
using System;
using System.Runtime.InteropServices;
public static class CcxNative {
  [DllImport("user32.dll", SetLastError=true, CharSet=CharSet.Unicode)]
  static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
  public static void Notify() {
    UIntPtr r;
    SendMessageTimeout((IntPtr)0xffff, 0x001A, UIntPtr.Zero, "Environment", 0x0002, 100, out r);
  }
}
'@
Add-Type -TypeDefinition $sig
[CcxNative]::Notify()
`;

export interface PersistResult {
  ok: boolean;
  error?: string;
}

/** 把一组受管键写进/删出 HKCU\Environment，并广播一次环境变更。 */
export function persistWindows(vals: EnvVals): PersistResult {
  const res = spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', PS_SCRIPT], {
    env: { ...process.env, CCX_ENV_PAYLOAD: JSON.stringify(vals) },
    stdio: ['ignore', 'ignore', 'pipe'],
    windowsHide: true,
  });
  if (res.error) return { ok: false, error: res.error.message };
  if (res.status !== 0) return { ok: false, error: (res.stderr?.toString() || '').trim() || `exit ${res.status}` };
  return { ok: true };
}
