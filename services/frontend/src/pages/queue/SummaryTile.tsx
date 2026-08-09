export function SummaryTile({
  label,
  value,
}: {
  label: string
  value: number
}) {
  return (
    <div className="rounded-2xl bg-white px-3 py-3.5 sm:p-[18px]">
      <div className="text-[11px] leading-tight text-avito-muted sm:text-sm line-clamp-2">
        {label}
      </div>
      <div className="mt-1.5 text-[22px] font-extrabold tabular-nums sm:mt-2 sm:text-[28px]">
        {value}
      </div>
    </div>
  )
}
