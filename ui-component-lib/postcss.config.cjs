module.exports = {
  plugins: {
    tailwindcss: {
      content: [
        '../console/consoleui/src/pages/**/*.tsx',
        '../console/consoleui/src/controls/**/*.tsx',
        '../console/consoleui/src/mainlayout/**/*.tsx',
        './src/**/*.{js,ts,jsx,tsx}',
        './pages/**/*.{js,ts,jsx,tsx}',
      ],
    },
    autoprefixer: {},
  },
};
