"use client"

import type React from "react"
import { useState } from "react"
import Link from "next/link"
import { Logo } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { SocialLoginButtons } from "@/components/social-login-buttons"

export default function LoginPage() {
  const [formData, setFormData] = useState({
    email: "",
    password: "",
    rememberMe: false,
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [success, setSuccess] = useState<Record<string, boolean>>({})

  const validateField = (name: string, value: string) => {
    switch (name) {
      case "email":
        if (!value) return "Email is required"
        if (!/\S+@\S+\.\S+/.test(value)) return "Email is invalid"
        return ""
      case "password":
        if (!value) return "Password is required"
        if (value.length < 6) return "Password must be at least 6 characters"
        return ""
      default:
        return ""
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))

    const error = validateField(name, value)
    setErrors((prev) => ({ ...prev, [name]: error }))
    setSuccess((prev) => ({ ...prev, [name]: !error && value.length > 0 }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const newErrors: Record<string, string> = {}
    Object.keys(formData).forEach((key) => {
      if (key !== "rememberMe") {
        const error = validateField(key, formData[key as keyof typeof formData] as string)
        if (error) newErrors[key] = error
      }
    })

    if (Object.keys(newErrors).length === 0) {
      console.log("Login successful", formData)
    } else {
      setErrors(newErrors)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">
      {/* Background pattern */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/10 via-background to-background" />
      <div
        className="absolute inset-0 opacity-30"
        style={{
          backgroundImage: `url('/images/auth-bg.jpg')`,
          backgroundSize: "cover",
          backgroundPosition: "center",
          filter: "blur(60px)",
        }}
      />

      {/* Login Card */}
      <div className="relative w-full max-w-md">
        <div className="bg-card border border-primary/20 rounded-xl p-8 card-glow backdrop-blur-sm">
          <div className="flex justify-center mb-8">
            <Logo />
          </div>

          <h1 className="text-3xl font-bold text-center mb-2 bg-linear-to-r from-primary via-primary/90 to-primary/70 bg-clip-text text-transparent">
            Welcome Back
          </h1>
          <p className="text-muted-foreground text-center mb-8">Sign in to book your next experience</p>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="email" className="text-foreground">
                Email
              </Label>
              <Input
                id="email"
                name="email"
                type="email"
                placeholder="you@example.com"
                value={formData.email}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.email ? "border-destructive focus:border-destructive" : ""
                } ${success.email ? "border-success focus:border-success" : ""}`}
              />
              {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password" className="text-foreground">
                Password
              </Label>
              <Input
                id="password"
                name="password"
                type="password"
                placeholder="••••••••"
                value={formData.password}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.password ? "border-destructive focus:border-destructive" : ""
                } ${success.password ? "border-success focus:border-success" : ""}`}
              />
              {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="remember"
                  checked={formData.rememberMe}
                  onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, rememberMe: checked as boolean }))}
                />
                <Label htmlFor="remember" className="text-sm text-muted-foreground cursor-pointer">
                  Remember me
                </Label>
              </div>
              <Link href="/forgot-password" className="text-sm text-primary hover:text-primary/80 transition-colors">
                Forgot password?
              </Link>
            </div>

            <Button
              type="submit"
              className="w-full bg-linear-to-r from-primary via-primary/90 to-primary/80 hover:from-primary/90 hover:via-primary/80 hover:to-primary/70 text-primary-foreground font-semibold shadow-lg shadow-primary/20 transition-all"
            >
              Sign In
            </Button>
          </form>

          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-border" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-card text-muted-foreground">Or continue with</span>
            </div>
          </div>

          <SocialLoginButtons />

          <p className="text-center text-sm text-muted-foreground mt-6">
            Don't have an account?{" "}
            <Link href="/register" className="text-primary hover:text-primary/80 font-semibold transition-colors">
              Create account
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
