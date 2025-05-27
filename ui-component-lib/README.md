# UserClouds UI

This repo contains the React component library for UserClouds and layout examples.

## About

This library is consumed by our user-facing UIs: [console](/console/consoleui/README.md) and [plex](/plex/plexui/).

### Storybook

The `src` folder contains the components and component dependencies.

- The React components are written in Typescript.
- Each component imports its own styles using SCSS Modules.
- Each component has its own Storybook file.
- Design tokens for the SCSS files — the variables for color, spacing, and type — are located in `./src/_variables.scss`.
- Icon components are located in `./src/icons/` and all icons are exported by `./src/icons/index.tsx`.
- The `./stories/` directory contains the long-form Storybook pages.
- The `./public/` directory is accessible by Storybook for images and other static assets.

#### Local development

```bash
yarn sb
```

A browser window should automatically opened to http://localhost:6006/.

Since we're using typescript we need to add `declarations.d.ts` so SCSS modules will work.

### Icons

The icons found in `src/icons/remix` are all taken from the open-source icon project [Remix Icons](https://remixicon.com/). You can, of course, add custom icons.

Instructions for adding new Remix icons are available in Storybook. The source file is `stories/Icons.stories.mdx`

### Using Tailwind

[Tailwind](https://tailwindcss.com/docs/installation) has been installed but is only being used sparingly in some files in the `pages` directory to simplify layout. For example, using the flexbox family of classes and some spacing classes. They are particularly helpful for [responsive layouts](https://github.com/tomgenoni/userclouds-ui/blob/main/pages/components/Cards/Authentication/Social/index.tsx#L46).

Tailwind has the additional benefit of only adding the classes that are used to the bundle. This means that instead of having a large CSS utility-class library load on the page, only the classes that are used in the code are included in the bundle. Tailwind accomplishes this by continually scanning for usage and including only those classes it finds.

Tailwind classes are **not** used in the core React files found in `src/components/` or `src/layouts`.

Color classes are also available as defined `tailwind.config.js` — for example, `text-neutral-600` — but they have not used in this project.

## Troubleshooting

Sometimes the Storybook UI gets into a weird state where the addon panel is hidden by local storage. In that case, running `localStorage.clear()` in the browser console and hard-reloading the page should fix it.
