import type { ComponentProps } from "react";
import { RouterProvider } from "react-router-dom";

type AppProvidersProps = Pick<ComponentProps<typeof RouterProvider>, "router">;

export function AppProviders({ router }: AppProvidersProps) {
  return <RouterProvider router={router} />;
}
