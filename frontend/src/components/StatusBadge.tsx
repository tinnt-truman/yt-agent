import type { AnalysisStatus } from '../types'

const LABELS: Record<AnalysisStatus, string> = {
  pending: 'Đang chờ',
  fetching: 'Đang lấy dữ liệu',
  analyzing: 'Đang phân tích',
  generating: 'Đang tạo chiến lược',
  done: 'Hoàn tất',
  failed: 'Lỗi',
}

const COLORS: Record<AnalysisStatus, string> = {
  pending: 'bg-slate-100 text-slate-700',
  fetching: 'bg-amber-100 text-amber-700',
  analyzing: 'bg-amber-100 text-amber-700',
  generating: 'bg-amber-100 text-amber-700',
  done: 'bg-emerald-100 text-emerald-700',
  failed: 'bg-red-100 text-red-700',
}

export function StatusBadge({ status }: { status: AnalysisStatus }) {
  return (
    <span className={`shrink-0 rounded-full px-3 py-1 text-xs font-medium ${COLORS[status]}`}>
      {LABELS[status]}
    </span>
  )
}
