import React from 'react';

interface LogoProps {
  className?: string;
  iconOnly?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

export const Logo: React.FC<LogoProps> = ({ className = '', iconOnly = false, size = 'md' }) => {
  const iconSizes = {
    sm: 'w-6 h-6',
    md: 'w-8 h-8',
    lg: 'w-10 h-10',
  };

  const textSizes = {
    sm: 'text-base',
    md: 'text-lg',
    lg: 'text-2xl',
  };

  return (
    <div className={`flex items-center gap-2.5 font-semibold tracking-tight ${className}`}>
      {/* Geometric Vault Shield Icon */}
      <div className={`relative flex items-center justify-center shrink-0 ${iconSizes[size]}`}>
        <svg viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full drop-shadow-[0_0_12px_rgba(118,103,245,0.4)]">
          {/* Hexagonal Outer Shield */}
          <path
            d="M16 2L28 8.5V20.5L16 27L4 20.5V8.5L16 2Z"
            stroke="url(#vault-gradient)"
            strokeWidth="2.2"
            strokeLinejoin="round"
          />
          {/* Internal Vault Nodes / AI Network */}
          <path
            d="M16 9L23 13V19L16 23L9 19V13L16 9Z"
            fill="url(#inner-gradient)"
            fillOpacity="0.25"
            stroke="#36C8D8"
            strokeWidth="1.4"
            strokeLinejoin="round"
          />
          <circle cx="16" cy="16" r="2.5" fill="#7667F5" />
          <defs>
            <linearGradient id="vault-gradient" x1="4" y1="2" x2="28" y2="27" gradientUnits="userSpaceOnUse">
              <stop stopColor="#7667F5" />
              <stop offset="1" stopColor="#36C8D8" />
            </linearGradient>
            <linearGradient id="inner-gradient" x1="9" y1="9" x2="23" y2="23" gradientUnits="userSpaceOnUse">
              <stop stopColor="#7667F5" />
              <stop offset="1" stopColor="#36C8D8" />
            </linearGradient>
          </defs>
        </svg>
      </div>

      {!iconOnly && (
        <span className={`text-[#F3F4F6] font-semibold tracking-tight flex items-center gap-1.5 ${textSizes[size]}`}>
          CloudVault <span className="text-[#36C8D8] font-bold">AI</span>
        </span>
      )}
    </div>
  );
};
