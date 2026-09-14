import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useUserContext } from "~/contexts/UserContext";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Payment complete");
}

export default function CheckoutSuccessPage() {
  const { syncUser, userInfo } = useUserContext();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let n = 0;
    const tick = async () => {
      await syncUser();
      n += 1;
      if (n >= 6) setReady(true);
    };
    void tick();
    const id = window.setInterval(() => {
      void tick();
    }, 1000);
    return () => window.clearInterval(id);
  }, [syncUser]);

  useEffect(() => {
    if (userInfo?.enrolled_courses.some((c) => c.has_access)) {
      setReady(true);
    }
  }, [userInfo]);

  return (
    <div className="w-full max-w-lg mx-auto px-4 py-16 text-center">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900 mb-4">
        Payment received
      </h1>
      <p className="text-slate-600 mb-6">
        {ready ? "Your access is ready." : "Confirming your access..."}
      </p>
      <Link to="/" className="text-primary-600 underline">
        Go to stories
      </Link>
    </div>
  );
}
