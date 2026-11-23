import * as React from "react";
import { cn } from "@/lib/utils";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  icon?: React.ReactNode;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, icon, ...props }, ref) => {
    return (
      <div className="relative w-full flex items-center">
        {icon && (
          <div className="absolute left-3 text-[#9296A3] pointer-events-none flex items-center">
            {icon}
          </div>
        )}
        <input
          type={type}
          className={cn(
            "flex h-10 w-full rounded-lg border border-[#242731] bg-[#191B22] px-3 py-2 text-sm text-[#F3F4F6] placeholder:text-[#9296A3] focus-visible:outline-none focus-visible:border-[#7667F5] focus-visible:ring-1 focus-visible:ring-[#7667F5] disabled:cursor-not-allowed disabled:opacity-50 transition-colors",
            icon && "pl-9",
            className
          )}
          ref={ref}
          {...props}
        />
      </div>
    );
  }
);
Input.displayName = "Input";

export { Input };
