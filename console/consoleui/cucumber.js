module.exports = {
  default: {
    import: ['features/**/*.ts'],
    format: ['progress', 'json:cucumber-report.json'],
    publishQuiet: true,
    failFast: true,
    backtrace: true,
    strict: true,
  },
};
