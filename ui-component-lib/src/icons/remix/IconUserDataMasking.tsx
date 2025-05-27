import { IconTypes, getSize } from '../iconHelpers';

function IconUserDataMasking({
  className,
  size = 'medium',
}: IconTypes): JSX.Element {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={getSize(size)}
      height={getSize(size)}
      className={className}
      fill="currentColor"
    >
      <title>User Data Masking</title>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M15.3 0H8.49997V1.7C5.68331 1.7 3.39997 3.98334 3.39997 6.8C3.39997 9.61666 5.68331 11.9 8.49997 11.9C11.3166 11.9 13.6 9.61666 13.6 6.8H15.3V0ZM13.6 6.8H12.6646L13.5273 5.93724C13.5751 6.2177 13.6 6.50595 13.6 6.8ZM8.49997 1.7C8.7684 1.7 9.032 1.72075 9.28922 1.7607L8.49997 2.55V1.7ZM8.49997 3.75206L10.2455 2.00651C10.5311 2.11047 10.8047 2.23934 11.0638 2.39031L8.49997 4.95411V3.75206ZM8.49997 6.15627L11.77 2.88618C11.9877 3.06828 12.19 3.26812 12.3746 3.48363L9.05825 6.8H8.49997V6.15627ZM10.2604 6.8L12.8779 4.18245C13.0314 4.43864 13.163 4.70945 13.2703 4.99219L11.4625 6.8H10.2604ZM0 15.5V18H17V15.5C17 14.1193 15.0083 13 13.6 13H3.4C1.99167 13 0 14.1193 0 15.5Z"
        fill="#283354"
      />{' '}
    </svg>
  );
}

export default IconUserDataMasking;
