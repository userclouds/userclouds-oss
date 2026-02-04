module.exports = {
  default: {
    requireModule: ['ts-node/register'],
    require: ['features/**/*.ts'],
    format: ['progress', 'json:cucumber-report.json'],
    publishQuiet: true,
    failFast: true,
    backtrace: true,
    strict: true,
  },
};
