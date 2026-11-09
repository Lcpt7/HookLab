import { useEffect, useRef } from "react";
import type { ReactNode } from "react";
import { X, Check, Copy, LoaderCircle } from "lucide-react";
import { useState } from "react";
export function Logo() {
  return (
    <div className="brand">
      <span className="brand-mark">
        <svg viewBox="0 0 32 32" fill="none">
          <path
            d="M9 7v18M23 7v18M9 16h14"
            stroke="currentColor"
            strokeWidth="3.5"
          />
          <circle cx="9" cy="7" r="3" fill="currentColor" />
          <circle cx="23" cy="25" r="3" fill="currentColor" />
        </svg>
      </span>
      <span>
        Hook<span className="brand-light">Lab</span>
        <sup>BETA</sup>
      </span>
    </div>
  );
}
export function Spinner() {
  return <LoaderCircle size={16} className="spin" />;
}
export function CopyButton({
  value,
  label = "复制",
  onError,
}: {
  value: string;
  label?: string;
  onError?: (s: string) => void;
}) {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (copied) {
      const t = setTimeout(() => setCopied(false), 1800);
      return () => clearTimeout(t);
    }
  }, [copied]);
  return (
    <button
      className="copy-button"
      title={label}
      aria-label={label}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value);
          setCopied(true);
        } catch {
          onError?.("复制失败，请手动选择并复制");
        }
      }}
    >
      {copied ? <Check size={15} /> : <Copy size={15} />}
      <span>{copied ? "已复制" : label}</span>
    </button>
  );
}
export function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    d?.showModal();
    return () => d?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className={"modal " + (wide ? "wide" : "")}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <header>
        <h2>{title}</h2>
        <button className="icon-button" onClick={onClose} aria-label="关闭弹窗">
          <X size={18} />
        </button>
      </header>
      {children}
    </dialog>
  );
}
export function Method({ value }: { value: string }) {
  return (
    <span className={"method method-" + value.toLowerCase()}>{value}</span>
  );
}
export function Code({ value }: { value: string }) {
  const tokens = value.split(
    /("(?:\\.|[^"\\])*"\s*:|"(?:\\.|[^"\\])*"|\b(?:true|false|null)\b|-?\b\d+(?:\.\d+)?\b)/g,
  );
  return (
    <code>
      {tokens.map((t, i) => (
        <span
          key={i}
          className={
            t.startsWith('"')
              ? t.trimEnd().endsWith(":")
                ? "json-key"
                : "json-string"
              : /^(true|false|null)$/.test(t)
                ? "json-bool"
                : /^-?\d/.test(t)
                  ? "json-number"
                  : ""
          }
        >
          {t}
        </span>
      ))}
    </code>
  );
}
