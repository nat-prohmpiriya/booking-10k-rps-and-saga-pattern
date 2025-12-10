import { Button } from "@/components/ui/button"
import { ArrowRight } from "lucide-react"
import Link from "next/link"

export function Hero() {
  return (
    <section className="relative h-[600px] lg:h-[700px] overflow-hidden">
      {/* Background Image with Gradient Overlay */}
      <div className="absolute inset-0">
        <img src="/images/hero-concert.jpg" alt="Hero event" className="w-full h-full object-cover" />
        <div className="absolute inset-0 bg-linear-to-t from-background via-background/70 to-background/30" />
      </div>

      {/* Content */}
      <div className="relative container mx-auto px-4 lg:px-8 h-full flex items-center">
        <div className="max-w-3xl space-y-6">
          <div className="inline-block glass px-4 py-2 rounded-full">
            <span className="text-primary text-sm font-medium">Premium Event Experiences</span>
          </div>
          <h1 className="text-4xl lg:text-7xl font-bold text-balance leading-tight">
            Discover & Book
            <span className="block bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
              Exclusive Events
            </span>
          </h1>
          <p className="text-lg lg:text-xl text-muted-foreground text-pretty max-w-2xl">
            Experience the finest concerts, festivals, and shows. Book your tickets to unforgettable moments.
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <Link href="/events">
              <Button
                size="lg"
                className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold"
              >
                Explore Events
                <ArrowRight className="ml-2 h-5 w-5" />
              </Button>
            </Link>
            <Button
              size="lg"
              variant="outline"
              className="border-primary text-primary hover:bg-primary/10 bg-transparent"
            >
              Learn More
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
