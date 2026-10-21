import {useQuery} from '@tanstack/react-query';
import {queryClient,queryOptions} from './queries';
export function useResource<T>(path:string,interval=0,enabled=true){
 const result=useQuery({...queryOptions<T>(path),enabled,refetchInterval:interval||false,refetchIntervalInBackground:false},queryClient);
 return {data:enabled?result.data:undefined,error:result.error?.message||'',reload:()=>{void result.refetch()}};
}
