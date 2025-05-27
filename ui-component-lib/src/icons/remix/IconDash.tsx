import { IconTypes } from '../iconHelpers';

function IconDash({ className }: IconTypes): JSX.Element {
  return (
    <svg
      width="15"
      height="3"
      viewBox="0 0 15 3"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <title> Dash Icon </title>
      <path
        d="M14.1669 0.666793V2.33321L0.833208 2.33321V0.666792L14.1669 0.666793Z"
        fill="#979797"
      />
    </svg>
  );
}

export default IconDash;
