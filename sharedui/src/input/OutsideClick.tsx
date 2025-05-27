import { useEffect, RefObject } from 'react';

// useOutsideClickDetector is a custom react hook to detect if the user clicks
// a location NOT contained within one of the provided element refs.
// This is useful for de-focusing or closing popups when a user clicks away.
function useOutsideClickDetector(
  refs: RefObject<HTMLDivElement>[],
  cb: () => void
): void {
  useEffect(() => {
    function onClick(e: Event) {
      let contained = false;
      refs.forEach((ref) => {
        if (ref.current && ref.current.contains(e.target as Node)) {
          contained = true;
        }
      });
      if (!contained) {
        cb();
      }
    }

    // Bind the event listener
    document.addEventListener('mousedown', onClick as unknown as EventListener);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener(
        'mousedown',
        onClick as unknown as EventListener
      );
    };
  }, [refs, cb]);
}

export default useOutsideClickDetector;
