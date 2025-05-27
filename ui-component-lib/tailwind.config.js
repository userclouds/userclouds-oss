/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    screens: {
      md: '768px',
      // => @media (min-width: 768px) { ... }
    },
    extend: {
      fontFamily: {
        sans: ['Inter'],
      },
    },
    colors: {
      transparent: 'transparent',
      current: 'currentColor',
      white: '#ffffff',
      black: '#0a0a0a',
    },
  },
  plugins: [],
};
