import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { googleSignIn } from "./actions";

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sponsorship Service</CardTitle>
          <CardDescription>
            Sign in with your Google Workspace account to access the admin
            dashboard.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form action={googleSignIn}>
            <Button type="submit" className="w-full">
              Sign in with Google
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
