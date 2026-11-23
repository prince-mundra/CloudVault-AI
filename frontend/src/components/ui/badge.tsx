import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-1 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-[#7667F5]/20 text-[#8679F7] border border-[#7667F5]/30",
        secondary:
          "border-transparent bg-[#191B22] text-[#9296A3] border border-[#242731]",
        cyan:
          "border-transparent bg-[#36C8D8]/15 text-[#36C8D8] border border-[#36C8D8]/30",
        success:
          "border-transparent bg-[#22C55E]/15 text-[#22C55E] border border-[#22C55E]/30",
        warning:
          "border-transparent bg-[#F59E0B]/15 text-[#F59E0B] border border-[#F59E0B]/30",
        destructive:
          "border-transparent bg-[#EF4444]/15 text-[#EF4444] border border-[#EF4444]/30",
        outline: "text-[#F3F4F6] border border-[#242731]",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
