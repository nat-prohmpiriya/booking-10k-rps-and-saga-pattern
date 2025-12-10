export function Logo({ className = "" }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <svg
        width="40"
        height="40"
        viewBox="0 0 40 40"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="text-primary"
      >
        <rect x="2" y="8" width="36" height="24" rx="4" stroke="currentColor" strokeWidth="2" />
        <path
          d="M10 16L16 22L26 12"
          stroke="currentColor"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle cx="32" cy="12" r="3" fill="currentColor" />
      </svg>
      <span className="text-2xl font-bold bg-linear-to-r from-primary via-primary/80 to-primary bg-clip-text text-transparent">
        BookingRush
      </span>
    </div>
  )
}
