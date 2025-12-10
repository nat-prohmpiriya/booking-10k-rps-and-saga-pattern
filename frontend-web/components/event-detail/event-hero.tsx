export function EventHero({ image }: { image?: string }) {
  return (
    <div className="relative h-[600px] w-full overflow-hidden">
      <img
        src={image || "/images/event-hero.jpg"}
        alt="Event"
        className="w-full h-full object-cover"
      />
      <div className="absolute inset-0 bg-linear-to-b from-transparent via-[#0a0a0a]/60 to-[#0a0a0a]" />
    </div>
  )
}
