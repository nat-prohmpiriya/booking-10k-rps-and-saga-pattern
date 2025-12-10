import { Calendar, Clock, MapPin } from "lucide-react"
import { Card } from "@/components/ui/card"

interface EventInfoProps {
  date?: string
  year?: string
  time?: string
  doorsOpen?: string
  venue?: string
  location?: string
}

export function EventInfo({
  date = "Saturday, March 15",
  year = "2025",
  time = "7:00 PM",
  doorsOpen = "6:30 PM",
  venue = "Grand Ballroom",
  location = "Crystal Palace Hotel"
}: EventInfoProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <Card className="bg-[#0f0f0f] border-[#1a1a1a] p-6">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-[#d4af37]/10 rounded-lg">
            <Calendar className="w-6 h-6 text-[#d4af37]" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground mb-1">Date</p>
            <p className="font-semibold text-lg">{date}</p>
            <p className="text-sm text-muted-foreground">{year}</p>
          </div>
        </div>
      </Card>

      <Card className="bg-[#0f0f0f] border-[#1a1a1a] p-6">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-[#d4af37]/10 rounded-lg">
            <Clock className="w-6 h-6 text-[#d4af37]" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground mb-1">Time</p>
            <p className="font-semibold text-lg">{time}</p>
            <p className="text-sm text-muted-foreground">Doors open at {doorsOpen}</p>
          </div>
        </div>
      </Card>

      <Card className="bg-[#0f0f0f] border-[#1a1a1a] p-6">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-[#d4af37]/10 rounded-lg">
            <MapPin className="w-6 h-6 text-[#d4af37]" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground mb-1">Venue</p>
            <p className="font-semibold text-lg">{venue}</p>
            <p className="text-sm text-muted-foreground">{location}</p>
          </div>
        </div>
      </Card>
    </div>
  )
}
