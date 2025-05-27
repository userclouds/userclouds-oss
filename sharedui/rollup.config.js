import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import { terser } from 'rollup-plugin-terser';
import postcss from 'rollup-plugin-postcss';

export default {
  input: 'src/index.tsx',
  output: {
    dir: 'build',
    format: 'esm',
    sourcemap: true,
  },
  external: ['react', 'react-dom', 'react-router'],
  plugins: [
    resolve(),
    commonjs({ extensions: ['.js', '.ts', '.jsx', 'tsx'] }),
    typescript({
      tsconfig: './tsconfig.json',
      sourceMap: true,
    }),
    terser(),
    postcss({
      modules: true,
    }),
  ],
};
