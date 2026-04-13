import { useState, useRef, useEffect } from "react";
import { Pencil, Trash2, Check, X } from "lucide-react";
import * as ContextMenu from "@radix-ui/react-context-menu";
import type { ConsoleMachine as Machine, ConsoleSession as Session } from "../../design-host/use-console-host";

interface ThreadItemProps {
  session: Session;
  machine: Machine;
  isSelected: boolean;
  onSelect: (machine: Machine, session: Session) => void;
  onRename?: (sessionId: string, newTitle: string) => void;
  onDelete?: (sessionId: string) => void;
}

const sessionStatusColor: Record<string, string> = {
  active: "bg-emerald-400",
  idle: "bg-zinc-600",
  completed: "bg-blue-400",
  systemError: "bg-red-400",
  notLoaded: "bg-amber-400",
  unknown: "bg-zinc-500",
};

export default function ThreadItem({
  session,
  machine,
  isSelected,
  onSelect,
  onRename,
  onDelete,
}: ThreadItemProps) {
  const [isRenaming, setIsRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState(session.title);
  const [swipeOffset, setSwipeOffset] = useState(0);
  const [isSwiping, setIsSwiping] = useState(false);
  const touchStartX = useRef(0);
  const touchCurrentX = useRef(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const SWIPE_THRESHOLD = 80;
  const MAX_SWIPE = 160;

  useEffect(() => {
    if (isRenaming && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isRenaming]);

  const handleRenameStart = () => {
    if (!onRename) {
      return;
    }
    setIsRenaming(true);
    setRenameValue(session.title);
  };

  const handleRenameConfirm = () => {
    if (renameValue.trim() && renameValue !== session.title) {
      onRename?.(session.id, renameValue.trim());
    }
    setIsRenaming(false);
  };

  const handleRenameCancel = () => {
    setIsRenaming(false);
    setRenameValue(session.title);
  };

  const handleDelete = () => {
    onDelete?.(session.id);
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    setIsSwiping(true);
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isSwiping) return;

    touchCurrentX.current = e.touches[0].clientX;
    const diff = touchStartX.current - touchCurrentX.current;
    const clampedDiff = Math.max(0, Math.min(diff, MAX_SWIPE));
    setSwipeOffset(clampedDiff);
  };

  const handleTouchEnd = () => {
    setIsSwiping(false);

    if (swipeOffset > SWIPE_THRESHOLD) {
      setSwipeOffset(MAX_SWIPE);
    } else {
      setSwipeOffset(0);
    }
  };

  const ThreadContent = (
    <div
      className="relative"
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
    >
      <div
        className="lg:hidden absolute right-0 top-0 bottom-0 flex items-center gap-1 pr-2"
        style={{
          width: `${MAX_SWIPE}px`,
          transform: `translateX(${MAX_SWIPE - swipeOffset}px)`,
          transition: isSwiping ? "none" : "transform 0.3s ease",
        }}
      >
        {onRename ? (
          <button
            onClick={handleRenameStart}
            className="flex items-center justify-center size-12 bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
          >
            <Pencil className="size-4 text-white" />
          </button>
        ) : null}
        {onDelete ? (
          <button
            onClick={handleDelete}
            className="flex items-center justify-center size-12 bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
          >
            <Trash2 className="size-4 text-white" />
          </button>
        ) : null}
      </div>

      <button
        onClick={() => {
          if (swipeOffset > 0) {
            setSwipeOffset(0);
          } else {
            onSelect(machine, session);
          }
        }}
        className={`w-full text-left flex items-start gap-2.5 px-2.5 py-2 rounded-lg transition-colors group relative ${
          isSelected
            ? "bg-zinc-800 text-zinc-100"
            : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50"
        }`}
        style={{
          transform: `translateX(-${swipeOffset}px)`,
          transition: isSwiping ? "none" : "transform 0.3s ease",
        }}
      >
        <div className="flex items-center gap-1.5 flex-shrink-0 mt-0.5">
          <div
            className={`size-1.5 rounded-full ${sessionStatusColor[session.status]} ${
              session.status === "active" ? "animate-pulse" : ""
            }`}
          />
        </div>

        {isRenaming ? (
          <div className="flex-1 flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
            <input
              ref={inputRef}
              type="text"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRenameConfirm();
                if (e.key === "Escape") handleRenameCancel();
              }}
              className="flex-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-100 focus:outline-none focus:border-blue-500"
            />
            {onRename ? (
              <>
                <button
                  onClick={handleRenameConfirm}
                  className="p-1 text-emerald-400 hover:bg-zinc-700 rounded transition-colors"
                >
                  <Check className="size-3.5" />
                </button>
                <button
                  onClick={handleRenameCancel}
                  className="p-1 text-zinc-500 hover:bg-zinc-700 rounded transition-colors"
                >
                  <X className="size-3.5" />
                </button>
              </>
            ) : null}
          </div>
        ) : (
          <div className="flex-1 min-w-0">
            <div className="text-xs leading-snug truncate">{session.title}</div>
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className="text-[10px] text-zinc-600 truncate">{session.agentName}</span>
              <span className="text-zinc-700 text-[10px]">·</span>
              <span className="text-[10px] text-zinc-600">{session.lastActivity}</span>
            </div>
          </div>
        )}
      </button>
    </div>
  );

  return onRename || onDelete ? (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>{ThreadContent}</ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content
          className="hidden lg:block min-w-[180px] bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1.5 z-50"
        >
          {onRename ? (
            <ContextMenu.Item
              onClick={handleRenameStart}
              className="flex items-center gap-2.5 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-700 hover:text-zinc-50 cursor-pointer outline-none"
            >
              <Pencil className="size-3.5" />
              <span>重命名</span>
            </ContextMenu.Item>
          ) : null}
          {onRename && onDelete ? (
            <ContextMenu.Separator className="h-px bg-zinc-700 my-1" />
          ) : null}
          {onDelete ? (
            <ContextMenu.Item
              onClick={handleDelete}
              className="flex items-center gap-2.5 px-3 py-2 text-xs text-red-400 hover:bg-zinc-700 hover:text-red-300 cursor-pointer outline-none"
            >
              <Trash2 className="size-3.5" />
              <span>删除</span>
            </ContextMenu.Item>
          ) : null}
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  ) : (
    ThreadContent
  );
}
