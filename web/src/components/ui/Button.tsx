import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Loader2 } from "lucide-react"
import { Slot } from "radix-ui"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 transition-transform duration-150 ease-out motion-safe:active:scale-[0.97]",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline:
          "border border-border bg-background hover:bg-accent hover:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground border border-border hover:bg-secondary/80",
        ghost:
          "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        xs: "h-6 gap-1 px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-8 gap-1.5 px-3 has-[>svg]:px-2.5",
        lg: "h-10 px-6 has-[>svg]:px-4",
        icon: "size-9",
        "icon-xs": "size-6 [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-8",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

type ButtonVariant = NonNullable<VariantProps<typeof buttonVariants>["variant"]>

const VARIANT_ALIAS: Record<string, ButtonVariant> = {
  primary: "default",
  danger: "destructive",
}

type VariantAlias = keyof typeof VARIANT_ALIAS

function Button({
  className,
  variant = "default",
  size = "default",
  fullWidth = false,
  loading = false,
  asChild = false,
  disabled,
  children,
  ...props
}: React.ComponentProps<"button"> &
  Omit<VariantProps<typeof buttonVariants>, "variant"> & {
    variant?: ButtonVariant | VariantAlias | null
    asChild?: boolean
    fullWidth?: boolean
    loading?: boolean
  }) {
  const resolvedVariant = VARIANT_ALIAS[variant as string] ?? variant
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={resolvedVariant}
      data-size={size}
      className={cn(
        buttonVariants({ variant: resolvedVariant, size }),
        fullWidth && "w-full",
        className
      )}
      disabled={disabled || loading}
      {...props}
    >
      {asChild ? (
        children
      ) : (
        <>
          {loading && <Loader2 className="size-4 animate-spin" />}
          {children}
        </>
      )}
    </Comp>
  )
}

export { Button, buttonVariants }
export type { ButtonVariant }
