import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded-lg text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[#7667F5] disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default:
          "bg-[#7667F5] text-white shadow-sm hover:bg-[#8679F7] active:bg-[#6858E8]",
        secondary:
          "bg-[#191B22] text-[#F3F4F6] border border-[#242731] hover:bg-[#1F222B] hover:border-[#323644]",
        ghost:
          "text-[#9296A3] hover:text-[#F3F4F6] hover:bg-[#191B22]",
        outline:
          "border border-[#242731] bg-transparent text-[#F3F4F6] hover:bg-[#191B22] hover:border-[#323644]",
        danger:
          "bg-[#EF4444]/10 text-[#EF4444] border border-[#EF4444]/20 hover:bg-[#EF4444]/20",
        cyan:
          "bg-[#36C8D8]/10 text-[#36C8D8] border border-[#36C8D8]/30 hover:bg-[#36C8D8]/20",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-lg px-3 text-xs",
        lg: "h-11 rounded-lg px-8 text-base",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
