import { QueryClient } from '@tanstack/react-query';
import { ApiError,request } from '../types';
// Only server responses live here. Unsaved edits belong to the editor and must
// never be replaced by a background refresh or stored as successful queries.
export const privateKey=(path:string)=>['private',path] as const;
export const queryClient=new QueryClient({defaultOptions:{queries:{staleTime:15_000,gcTime:60_000,refetchOnWindowFocus:true,refetchOnReconnect:true,retry:(count,error)=>count<2&&(!(error instanceof ApiError)||error.status>=500),retryDelay:attempt=>Math.min(1000*2**attempt,5000)}}});
export const queryOptions=<T,>(path:string)=>({queryKey:privateKey(path),queryFn:({signal}:{signal:AbortSignal})=>request<T>(path,{signal})});
export function clearPrivateQueries(){void queryClient.cancelQueries();queryClient.clear()}
export function invalidateWorkspace(){void queryClient.invalidateQueries({predicate:query=>query.queryKey[0]==='private'&&query.queryKey[1]!=='/api/v1/auth/session'})}
if(typeof window!=='undefined'){
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)void queryClient.refetchQueries({type:'active'},{cancelRefetch:false})});
 window.addEventListener('online',()=>{void queryClient.refetchQueries({type:'active'},{cancelRefetch:false})});
 window.addEventListener('glance:unauthorized',clearPrivateQueries);
 window.addEventListener('glance:logout',clearPrivateQueries);
 window.addEventListener('glance:mutated',invalidateWorkspace);
 window.addEventListener('glance:refresh',()=>{void queryClient.invalidateQueries({predicate:query=>String(query.queryKey[1]).endsWith('/snapshots')})});
}
