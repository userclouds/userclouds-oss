import { IconTypes, getSize } from '../iconHelpers';

function IconUserDataStorage({
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
      <title>User Data Storage</title>
      <path
        d="M17.5 7.91667V10.4167C17.5 12.4875 14.1417 14.1667 10 14.1667C5.85833 14.1667 2.5 12.4875 2.5 10.4167V7.91667C2.5 9.9875 5.85833 11.6667 10 11.6667C14.1417 11.6667 17.5 9.9875 17.5 7.91667ZM2.5 12.0833C2.5 14.1542 5.85833 15.8333 10 15.8333C14.1417 15.8333 17.5 14.1542 17.5 12.0833V14.5833C17.5 16.6542 14.1417 18.3333 10 18.3333C5.85833 18.3333 2.5 16.6542 2.5 14.5833V12.0833ZM10 10C5.85833 10 2.5 8.32083 2.5 6.25C2.5 4.17917 5.85833 2.5 10 2.5C14.1417 2.5 17.5 4.17917 17.5 6.25C17.5 8.32083 14.1417 10 10 10Z"
        fill="#283354"
      />
    </svg>
  );
}

export default IconUserDataStorage;
