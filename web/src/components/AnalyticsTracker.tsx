"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { trackEvent } from "@/lib/analytics";

export function AnalyticsTracker() {
  const pathname = usePathname();

  useEffect(() => {
    trackEvent({ event: "registry_visit", page: pathname });
  }, [pathname]);

  return null;
}
