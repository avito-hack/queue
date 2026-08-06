export function SummaryTile({
  label,
  value,
}: {
  label: string
  value: number
}) {
  return (
    <div className="rounded-2xl bg-white p-[18px]">
      <div className="text-sm text-avito-muted">{label}</div>
      <div className="mt-2 text-[28px] font-extrabold">{value}</div>
    </div>
  )
}
