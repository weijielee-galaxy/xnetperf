ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20000 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20001 &
ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20002 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20003 &
ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20004 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20005 &
