import { IconTypes, getSize } from '../iconHelpers';

function IconUserAuthentication({
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
      <title>User Authentication</title>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M2 0C0.895431 0 0 0.895431 0 2V17C0 18.1046 0.89543 19 2 19H16C17.1046 19 18 18.1046 18 17V2C18 0.895431 17.1046 0 16 0H2ZM6 1H2C1 1 1 2 1 2V6H2V2.5C2 2.5 2 2 2.5 2H6V1ZM12 1H16C17 1 17 2 17 2V6H16V2.5C16 2.5 16 2 15.5 2H12V1ZM6 18H2C1 18 1 17 1 17V13H2V16.5C2 16.5 2 17 2.5 17H6V18ZM12 17V18H16C17 18 17 17 17 17V13H16V16.5C16 16.5 16 17 15.5 17H12ZM8.82928 10C8.41748 11.1652 7.30621 12 6 12C4.34314 12 3 10.6569 3 9C3 7.34314 4.34314 6 6 6C7.30621 6 8.41748 6.83484 8.82928 8H14.1875C15.1875 8 15.1875 9 15.1875 9V12H14V11H13V12H12V10.5C12 10.5 12 10 11.5 10H8.82928ZM7 9C7 9.55228 6.55228 10 6 10C5.44772 10 5 9.55228 5 9C5 8.44772 5.44772 8 6 8C6.55228 8 7 8.44772 7 9Z"
        fill="#283354"
      />
    </svg>
  );
}

export default IconUserAuthentication;
