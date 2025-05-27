import { IconTypes, getSize } from '../iconHelpers';

function IconMenuHome({ className, size = 'medium' }: IconTypes): JSX.Element {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={getSize(size)}
      height={getSize(size)}
      className={className}
      fill="currentColor"
    >
      <title>Menu Home</title>
      <path
        d="M3.13158 18H7.97368C8.24983 18 8.47368 17.7761 8.47368 17.5V12.5C8.47368 12.2239 8.69754 12 8.97368 12H12.0263C12.3025 12 12.5263 12.2239 12.5263 12.5V17.5C12.5263 17.7761 12.7502 18 13.0263 18H17.8684C18.1446 18 18.3684 17.7761 18.3684 17.5V9C18.3684 8.72386 18.5923 8.5 18.8684 8.5H20.4823C20.9646 8.5 21.1674 7.88457 20.7795 7.59791L10.7972 0.219666C10.6206 0.089117 10.3794 0.089117 10.2028 0.219666L0.220472 7.59791C-0.167365 7.88457 0.0353892 8.5 0.517668 8.5H2.13158C2.40772 8.5 2.63158 8.72386 2.63158 9V17.5C2.63158 17.7761 2.85544 18 3.13158 18Z"
        fill="#283354"
      />
    </svg>
  );
}

export default IconMenuHome;
