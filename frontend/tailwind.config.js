/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        canvas: '#101114',
        surface: '#15171D',
        card: {
          DEFAULT: '#191B22',
          hover: '#1F222B',
        },
        hairline: {
          DEFAULT: '#242731',
          hover: '#323644',
        },
        primary: {
          DEFAULT: '#7667F5',
          hover: '#8679F7',
        },
        secondary: {
          DEFAULT: '#36C8D8',
        },
        text: {
          primary: '#F3F4F6',
          muted: '#9296A3',
        },
        status: {
          success: '#22C55E',
          warning: '#F59E0B',
          danger: '#EF4444',
        },
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
      },
      borderRadius: {
        xl: '12px',
        lg: '8px',
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [],
};
