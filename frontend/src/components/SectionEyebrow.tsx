export function SectionEyebrow({
  icon,
  children,
  accent = false,
}: {
  icon: React.ReactNode
  children: React.ReactNode
  accent?: boolean
}) {
  return (
    <div className="mb-3 flex items-center gap-2">
      {icon}
      <span
        className={`text-xs font-semibold uppercase tracking-wider ${
          accent ? 'text-violet-700' : 'text-slate-500'
        }`}
      >
        {children}
      </span>
    </div>
  )
}

export function AnalysisIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="text-slate-500"
    >
      <rect x="3" y="3" width="18" height="18" rx="3" />
      <path d="M9 9h6v6H9z" />
    </svg>
  )
}

export function SparkleIcon({ className = '' }: { className?: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" className={className}>
      <path d="M12 2l1.8 5.6L19 9l-5.2 1.4L12 16l-1.8-5.6L5 9l5.2-1.4L12 2z" />
    </svg>
  )
}
