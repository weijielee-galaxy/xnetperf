ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20000  10.155.84.3 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20001  10.155.84.3 &
