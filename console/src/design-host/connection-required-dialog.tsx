interface ConnectionRequiredDialogProps {
  open: boolean;
  message: string;
  onOpenSettings: () => void;
}

export function ConnectionRequiredDialog(props: ConnectionRequiredDialogProps) {
  if (!props.open) {
    return null;
  }

  return (
    <div
      className="absolute inset-0 z-[200] flex items-center justify-center bg-black/60 px-6"
      role="dialog"
      aria-modal="true"
      aria-label="Gateway 连接未配置"
    >
      <div className="w-full max-w-md rounded-xl border border-zinc-700 bg-zinc-900 p-6">
        <h2 className="text-lg text-zinc-100">Gateway 连接未配置</h2>
        <p className="mt-3 text-sm text-zinc-400">{props.message}</p>
        <button
          type="button"
          onClick={props.onOpenSettings}
          className="mt-5 rounded-lg bg-blue-600 px-4 py-2 text-sm text-white transition-colors hover:bg-blue-500"
        >
          前往设置
        </button>
      </div>
    </div>
  );
}
