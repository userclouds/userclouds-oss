import { pathToRegexp, Key } from 'path-to-regexp';

type RouteArrayEntry = {
  path: string;
  handler: Function;
};
type RouteMapEntry = {
  routeParts: Key[];
  handler: Function;
  exp: RegExp;
};
type RouteMap = Record<string, RouteMapEntry>;

type RouteMatch = {
  pattern: string;
  handler: Function;
  params: Record<string, string>;
};

const matchToParams = (routeParts: Key[], match: string[]) => {
  return routeParts.reduce(
    (acc: Record<string, string>, part: Key, i: number) => {
      acc[part.name] = match[i + 1];
      return acc;
    },
    {}
  );
};

const getRouteMatchScore = (pattern: string, url: URL) => {
  let matches = 0;
  const patternParts = pattern.split('/');
  const urlParts = url.pathname.split('/');
  if (patternParts.length !== urlParts.length) {
    return 0;
  }
  for (let i = 0; i < patternParts.length; i++) {
    if (patternParts[i] === urlParts[i]) {
      matches++;
    }
  }
  return matches;
};

const matchURLToRoute =
  (map: RouteMap) =>
  (url: URL): RouteMatch | undefined => {
    let match: string[] | null;
    let routeMatch: RouteMatch | undefined;
    for (const pattern in map) {
      const route: RouteMapEntry = map[pattern];
      match = url.pathname.match(route.exp);

      if (!match) {
        continue;
      }

      const params: Record<string, string> = matchToParams(
        route.routeParts,
        match
      );

      if (
        !routeMatch ||
        getRouteMatchScore(pattern, url) >
          getRouteMatchScore(routeMatch.pattern, url)
      ) {
        routeMatch = {
          pattern,
          handler: route.handler,
          params,
        };
      }
    }
    return routeMatch;
  };

const routeMatcher = (routes: RouteArrayEntry[]) => {
  const map: RouteMap = {};

  for (const route of routes) {
    const routeParts: Key[] = [];
    const { handler, path } = route;
    const pattern = path;
    map[pattern] = {
      routeParts, // starts out empty but gets modified by the pathToRegexp call
      handler,
      exp: pathToRegexp(pattern, routeParts),
    };
  }

  return {
    routes,
    match: matchURLToRoute(map),
  };
};

export default routeMatcher;
