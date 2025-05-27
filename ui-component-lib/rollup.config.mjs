import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import image from '@rollup/plugin-image';
import { terser } from 'rollup-plugin-terser';
import postcss from 'rollup-plugin-postcss';
import nodePolyfills from 'rollup-plugin-polyfill-node';
import jsx from 'acorn-jsx';

const ignoreWarnings = ['THIS_IS_UNDEFINED', 'MODULE_LEVEL_DIRECTIVE'];

export default {
  input: 'src/index.ts',
  output: {
    dir: 'build',
    format: 'esm',
    sourcemap: true,
  },
  onwarn: (warning) => {
    // DateTimePicker presents this warning but we can safely ignore
    if (ignoreWarnings.includes(warning.code)) {
      return;
    }

    // eslint-disable-next-line no-console
    console.warn(warning.message);
  },
  external: ['react', 'react-dom'],
  acornInjectPlugins: [jsx()],
  plugins: [
    resolve({
      browser: true,
      preferBuiltins: false,
    }),
    image(),
    commonjs({ extensions: ['.js', '.ts', '.jsx', '.tsx'] }),
    typescript({
      tsconfig: './tsconfig.rollup.json',
    }),
    nodePolyfills(),
    terser(),
    postcss({
      extract: false,
      modules: true,
      use: [
        [
          'sass',
          {
            verbose: false,
            // Fix for Dart Sass 2.0.0 deprecation warning
            silenceDeprecations: ['legacy-js-api'],
          },
        ],
      ],
    }),
  ],
};
