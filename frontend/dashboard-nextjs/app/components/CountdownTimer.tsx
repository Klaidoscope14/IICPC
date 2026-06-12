'use client';

import { useState, useEffect } from 'react';

export default function CountdownTimer() {
  const [targetDate, setTargetDate] = useState<string | null>(null);
  const [timeLeft, setTimeLeft] = useState<{
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
  } | null>(null);

  useEffect(() => {
    fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}/api/v1/admin/hackathon/dates`)
      .then(res => res.json())
      .then(data => {
        if (data && data.end_date) {
          setTargetDate(data.end_date);
        }
      })
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (!targetDate) return;

    const calculateTimeLeft = () => {
      const difference = new Date(targetDate).getTime() - new Date().getTime();
      
      if (difference > 0) {
        setTimeLeft({
          days: Math.floor(difference / (1000 * 60 * 60 * 24)),
          hours: Math.floor((difference / (1000 * 60 * 60)) % 24),
          minutes: Math.floor((difference / 1000 / 60) % 60),
          seconds: Math.floor((difference / 1000) % 60)
        });
      } else {
        setTimeLeft(null);
      }
    };

    calculateTimeLeft();
    const timer = setInterval(calculateTimeLeft, 1000);

    return () => clearInterval(timer);
  }, [targetDate]);

  if (!timeLeft) {
    return <div className="text-xl font-bold text-gray-500">Hackathon has ended.</div>;
  }

  return (
    <div className="flex gap-4 text-center">
      <div className="flex flex-col bg-gray-800 p-4 rounded-lg shadow min-w-[80px]">
        <span className="text-3xl font-bold text-blue-400">{timeLeft.days}</span>
        <span className="text-sm text-gray-400 uppercase tracking-wider">Days</span>
      </div>
      <div className="flex flex-col bg-gray-800 p-4 rounded-lg shadow min-w-[80px]">
        <span className="text-3xl font-bold text-blue-400">{timeLeft.hours}</span>
        <span className="text-sm text-gray-400 uppercase tracking-wider">Hours</span>
      </div>
      <div className="flex flex-col bg-gray-800 p-4 rounded-lg shadow min-w-[80px]">
        <span className="text-3xl font-bold text-blue-400">{timeLeft.minutes}</span>
        <span className="text-sm text-gray-400 uppercase tracking-wider">Mins</span>
      </div>
      <div className="flex flex-col bg-gray-800 p-4 rounded-lg shadow min-w-[80px]">
        <span className="text-3xl font-bold text-blue-400">{timeLeft.seconds}</span>
        <span className="text-sm text-gray-400 uppercase tracking-wider">Secs</span>
      </div>
    </div>
  );
}
