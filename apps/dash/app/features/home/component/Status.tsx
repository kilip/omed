import { useAuth } from "~/shared/providers/AuthProvider";

export default function Status() {
  const { user } = useAuth();
  return (
    <div>
      {user.name} = {user.image}
    </div>
  );
}
