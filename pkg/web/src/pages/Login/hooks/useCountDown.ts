import { useState, useEffect, useRef } from 'react';

const useCountdown = (duration: number) => {
  const [countdown, setCountdown] = useState(0);
  const [isSetup, toggleSetup] = useState(false);
  const timer = useRef<NodeJS.Timeout>();

  useEffect(() => {
    if (isSetup) {
      setCountdown(duration);
      timer.current = setInterval(() => {
        setCountdown((current) => current - 1);
      }, 1000);
    } else clearInterval(timer.current as NodeJS.Timeout);
  }, [isSetup]);

  useEffect(() => {
    if (countdown === 0) {
      toggleSetup(false);
    }
  }, [countdown]);

  useEffect(() => () => clearInterval(timer.current as NodeJS.Timeout), []);

  function setupCountdown() {
    toggleSetup(true);
  }

  return { countdown, setupCountdown };
};

export default useCountdown;
