"use client"

import type React from "react"
import { useState } from "react"
import Link from "next/link"
import { Logo } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { SocialLoginButtons } from "@/components/social-login-buttons"

export default function RegisterPage() {
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    phone: "",
    password: "",
    confirmPassword: "",
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [success, setSuccess] = useState<Record<string, boolean>>({})

  const validateField = (name: string, value: string) => {
    switch (name) {
      case "name":
        if (!value) return "Name is required"
        if (value.length < 2) return "Name must be at least 2 characters"
        return ""
      case "email":
        if (!value) return "Email is required"
        if (!/\S+@\S+\.\S+/.test(value)) return "Email is invalid"
        return ""
      case "phone":
        if (!value) return "Phone is required"
        if (!/^\+?[\d\s-()]+$/.test(value)) return "Phone number is invalid"
        return ""
      case "password":
        if (!value) return "Password is required"
        if (value.length < 8) return "Password must be at least 8 characters"
        if (!/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(value))
          return "Password must contain uppercase, lowercase, and number"
        return ""
      case "confirmPassword":
        if (!value) return "Please confirm your password"
        if (value !== formData.password) return "Passwords do not match"
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

    // Re-validate confirm password if password changes
    if (name === "password" && formData.confirmPassword) {
      const confirmError = validateField("confirmPassword", formData.confirmPassword)
      setErrors((prev) => ({ ...prev, confirmPassword: confirmError }))
      setSuccess((prev) => ({ ...prev, confirmPassword: !confirmError }))
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const newErrors: Record<string, string> = {}
    Object.keys(formData).forEach((key) => {
      const error = validateField(key, formData[key as keyof typeof formData])
      if (error) newErrors[key] = error
    })

    if (Object.keys(newErrors).length === 0) {
      console.log("Registration successful", formData)
    } else {
      setErrors(newErrors)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">
      {/* Background pattern */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom,_var(--tw-gradient-stops))] from-primary/10 via-background to-background" />
      <div
        className="absolute inset-0 opacity-30"
        style={{
          backgroundImage: `url('/images/auth-bg-2.jpg')`,
          backgroundSize: "cover",
          backgroundPosition: "center",
          filter: "blur(60px)",
        }}
      />

      {/* Register Card */}
      <div className="relative w-full max-w-md">
        <div className="bg-card border border-primary/20 rounded-xl p-8 card-glow backdrop-blur-sm">
          <div className="flex justify-center mb-8">
            <Logo />
          </div>

          <h1 className="text-3xl font-bold text-center mb-2 bg-linear-to-r from-primary via-primary/90 to-primary/70 bg-clip-text text-transparent">
            Create Account
          </h1>
          <p className="text-muted-foreground text-center mb-8">Join us and start booking amazing events</p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name" className="text-foreground">
                Full Name
              </Label>
              <Input
                id="name"
                name="name"
                type="text"
                placeholder="John Doe"
                value={formData.name}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.name ? "border-destructive focus:border-destructive" : ""
                } ${success.name ? "border-success focus:border-success" : ""}`}
              />
              {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
            </div>

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
              <Label htmlFor="phone" className="text-foreground">
                Phone Number
              </Label>
              <Input
                id="phone"
                name="phone"
                type="tel"
                placeholder="+66 81 234 5678"
                value={formData.phone}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.phone ? "border-destructive focus:border-destructive" : ""
                } ${success.phone ? "border-success focus:border-success" : ""}`}
              />
              {errors.phone && <p className="text-sm text-destructive">{errors.phone}</p>}
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

            <div className="space-y-2">
              <Label htmlFor="confirmPassword" className="text-foreground">
                Confirm Password
              </Label>
              <Input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                placeholder="••••••••"
                value={formData.confirmPassword}
                onChange={handleChange}
                className={`bg-secondary border-border focus:border-primary transition-all ${
                  errors.confirmPassword ? "border-destructive focus:border-destructive" : ""
                } ${success.confirmPassword ? "border-success focus:border-success" : ""}`}
              />
              {errors.confirmPassword && <p className="text-sm text-destructive">{errors.confirmPassword}</p>}
            </div>

            <Button
              type="submit"
              className="w-full bg-linear-to-r from-primary via-primary/90 to-primary/80 hover:from-primary/90 hover:via-primary/80 hover:to-primary/70 text-primary-foreground font-semibold shadow-lg shadow-primary/20 transition-all"
            >
              Create Account
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
            Already have an account?{" "}
            <Link href="/login" className="text-primary hover:text-primary/80 font-semibold transition-colors">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
