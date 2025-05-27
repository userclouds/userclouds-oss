import { IconTypes } from '../iconHelpers';

function IconCheckmark({ className }: IconTypes): JSX.Element {
  return (
    <svg
      width="15"
      height="11"
      viewBox="0 0 15 11"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <title> Checkmark Icon </title>
      <path
        d="M5.30333 7.66083L12.9633 0L14.1425 1.17833L5.30333 10.0175L0 4.71417L1.17833 3.53583L5.30333 7.66083Z"
        fill="#199C16"
      />
    </svg>
  );
}

export default IconCheckmark;
